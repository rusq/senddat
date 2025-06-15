#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Generate a CSV file with ESC/POS commands from the Epson website.
"""
import escpos.escpos as escpos
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def fetch_all(index: list[escpos.IndexEntry]) -> None:
    """
    Fetch all commands from the index and save them to the cache.
    """
    for entry in index:
        cmd = escpos.parse_url_loc(entry.url)
        logger.info(f"Parsed command: {cmd.title} - {cmd.name}: {cmd.format}")

def debug_command():
    for file in [ "gs_lparen_ck_fn48.html" ]:
        cmd = escpos.parse_url_loc(file)
        logger.info(f"Parsed command from file: {cmd.title} - {cmd.name}: {cmd.format}")


if __name__ == "__main__":
    version = escpos.version()
    logger.info(f"ESC/POS version: {version}")
    index = escpos.command_index()
    logger.info(f"Found {len(index)} commands")
    for entry in index:
        print("%-15s - %s (%s)" % (entry.code, entry.name, entry.url))
    logger.info("Parsing commands...")
    fetch_all(index)
