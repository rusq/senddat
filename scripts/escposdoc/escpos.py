# -*- coding: utf-8 -*-
import logging
import os
import shutil
import urllib.request

from dataclasses import dataclass
from bs4 import BeautifulSoup

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

URL_BASE = "https://download4.epson.biz/sec_pubs/pos/reference_en/escpos/"
_USER_AGENT = "Links (2.30; Darwin 23.6.0 x86_64; LLVM/Clang 15.0; text)"

_C_VERSION_FILE = "index.html"
_C_COMMANDS_FILE = "commands.html"

CACHE_DIR = "cache"
if not os.path.exists(CACHE_DIR):
    os.makedirs(CACHE_DIR)


def url_of(html: str) -> str:
    return URL_BASE + html


def cachefile(filename: str) -> str:
    return os.path.join(CACHE_DIR, filename)


@dataclass
class IndexEntry:
    code: str
    name: str
    url: str

    def __str__(self):
        return f"{self.code} - {self.name} ({self.url})"


def _get_file(url: str, dest: str) -> str:
    req = urllib.request.Request(url)
    req.add_header(
        "User-Agent", _USER_AGENT
    )
    with open(dest, "wb") as w:
        with urllib.request.urlopen(req) as response:
            try:
                shutil.copyfileobj(response, w)
            except Exception as e:
                os.unlink(dest)
                raise
    with open(dest, "rt") as r:
        return r.read()


def _load_or_fetch_file(url: str, dest: str):
    """Load a file from the cache or fetch it from the URL if it doesn't exist."""
    if os.path.exists(dest):
        with open(dest, "rt") as r:
            return r.read()
    return _get_file(url, dest)


def open_doc(filename: str) -> str:
    return _load_or_fetch_file(url_of(filename), cachefile(filename))


def command_index() -> list[IndexEntry]:
    """
    Parse the command index and return a list of IndexEntry objects.

    The entry looks like this:
    <tr>
    <td align="left" nowrap=""><a href="esc_lr.html">ESC r</a></td>
    <td align="left">Select print color</td>
    <td align="left" nowrap="">Character</td>
    </tr>
    """

    doc = BeautifulSoup(open_doc(_C_COMMANDS_FILE), features="html.parser")
    tbl = doc.find("table")
    if not tbl:
        raise ValueError("Unable to find the table")
    body = tbl.find("tbody")
    if not body:
        raise ValueError("No table body")
    commands = []
    for row in body.find_all("tr"):
        cols = row.find_all('td')
        if len(cols) != 3:
            raise ValueError(f"invalid number of columns in {row.prettify()}")
        code = cols[0].text
        href = cols[0].find('a').attrs.get('href')
        namelines = [line.strip()
                     for line in cols[1].text.splitlines() if line.strip()]
        name = " ".join(namelines)
        commands.append(IndexEntry(code=code, name=name, url=href))
    return commands


def version() -> str:
    """
    Get the version of the ESC/POS commands.

    The version is in the index.html file, which looks like this:
    <h2 class="Head-C" id="QAccess0">ESC/POS<sup>®</sup> Command Reference Revision 3.40
                                 </h2>
    """
    doc = BeautifulSoup(open_doc(_C_VERSION_FILE), features="html.parser")
    h2 = doc.find("h2", class_="Head-C")
    if not h2:
        raise ValueError("Unable to find the version header")
    text = h2.text.strip()
    if not text:
        raise ValueError("Version header is empty")
    # The version is the last part of the text, after the last space
    version = text.split()[-1]
    if not version:
        raise ValueError("Version is empty")
    return version

@dataclass
class CommandNotation:
    prefix: list[str]
    arguments: list[str]
    payload: list[str]
    parameters: bool = False

    def __str__(self) -> str:
        return f"Prefix: {self.prefix}, Arguments: {self.arguments}, Payload: {self.payload}, Parameters: {self.parameters}"

    

@dataclass
class CommandFormat:
    """
    Command format for ESC/POS commands.
    each member is a tuple of ( prefix, parameters, payload )
    prefix and parameters are lists of strings.
    The format is parsed from the command documentation.
    """
    ascii: CommandNotation
    hex: CommandNotation
    decimal: CommandNotation

    def __str__(self) -> str:
        return self.hex.__str__()

    @classmethod
    def parse(cls, doc: BeautifulSoup) -> 'Command':
        tbl_params = doc.find("table", class_="parameter")
        if not tbl_params:
            raise ValueError(f"Unable to find the parameter table")
        rows = tbl_params.find_all("tr")
        if not rows:
            raise ValueError(f"No parameters found")

        row_map = [
            CommandNotation([], [], [], False),  # ASCII
            CommandNotation([], [], [], False),  # Hex
            CommandNotation([], [], [], False),  # Decimal
        ]
        for i, row in enumerate(rows):
            for col in rows[i].find_all("td")[1:]:
                # parameters will have a special font class: <div><font class="parameter">m</font></div>
                if col.text.strip() == "":
                    continue
                if text := col.text.strip():
                    if col.find("font", class_="parameter"):
                        if text == "[parameters]":
                            row_map[i].parameters = True
                        elif "..." in text:
                            row_map[i].payload.append(text)
                        else:
                            row_map[i].arguments.append(text)
                    else:
                        row_map[i].prefix.append(text)
        return cls(ascii=row_map[0], hex=row_map[1], decimal=row_map[2])


@dataclass
class Command:
    title: str
    name: str
    format: CommandFormat = None

    @classmethod
    def parse(cls, doc: BeautifulSoup) -> 'Command':
        title = doc.find("h1", class_="Head-B").text.strip()
        name = doc.select_one(
            "#body-contents > div > div:nth-child(3) > div > div > div").text.strip()
        format = CommandFormat.parse(doc)
        return cls(title=title, name=name, format=format)


def parse_command(filename: str) -> Command:
    html = open_doc(filename)
    doc = BeautifulSoup(html, features="html.parser")
    if not doc:
        raise ValueError(f"Unable to parse the document {filename}")

    return Command.parse(doc)
