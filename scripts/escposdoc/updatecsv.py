#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Generate a CSV file with ESC/POS commands from the Epson website.
"""
import escpos
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


if __name__ == "__main__":
    version = escpos.version()
    logger.info(f"ESC/POS version: {version}")
    index = escpos.command_index()
    logger.info(f"Found {len(index)} commands")
    for entry in index:
        print("%-15s - %s (%s)" % (entry.code, entry.name, entry.url))
    for cmd in [
        "esc_asterisk.html",
        "gs_cd.html",
        "lf.html",
        "esc_2.html",
        "esc_3.html",
        "dle_dc4_fn1.html",
        "gs_lparen_lk.html",
        "gs_lparen_lk_fn281.html",
    ]:
        cmd = escpos.parse_command(cmd)
        logger.info(f"Parsed command: {cmd.title} - {cmd.name}: {cmd.format}")
