# -*- coding: utf-8 -*-
import test_fixtures
import unittest

from escpos import CommandFormat, CommandNotation
from bs4 import BeautifulSoup


class TestCommandFormat(unittest.TestCase):

    def test_parse(self):
        doc = BeautifulSoup(test_fixtures.CMD_FMT_ESC_STAR,
                            features="html.parser")
        cmd = CommandFormat.parse(doc)

        self.assertEqual(cmd.ascii, CommandNotation(
            prefix=['ESC', '*'],
            arguments=['m', 'nL', 'nH'],
            payload=['d1 ... dk']
        ))
        self.assertEqual(cmd.hex, CommandNotation(
            prefix=['1B', '2A'],
            arguments=['m', 'nL', 'nH'],
            payload=['d1 ... dk']
        ))
        self.assertEqual(cmd.decimal, CommandNotation(
            prefix=['27', '42'],
            arguments=['m', 'nL', 'nH'],
            payload=['d1 ... dk']
        ))

    def test_parse_gs_v(self):
        doc = BeautifulSoup(test_fixtures.CMD_FMT_GS_V,
                            features="html.parser")
        cmd = CommandFormat.parse(doc)

    def test_parse_gs_lparen_e_fn51(self):
        doc = BeautifulSoup(test_fixtures.CMD_FMT_GS_LPAREN_E_FN_51,
                            features="html.parser")
        cmd = CommandFormat.parse(doc)
