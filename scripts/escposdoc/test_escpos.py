# -*- coding: utf-8 -*-
import unittest

TEST_ESC_STAR = """<div xmlns="" class="Header2">
                                 <h2 class="Head-C" id="QAccess1">[Format]</h2>
                                 <div class="indent">
                                    <div>
                                       <table class="parameter">
                                          <tbody>
                                             <tr>
                                                <td style="text-align:left;">
                                                   <div>
                                                      <div>ASCII</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>&nbsp;&nbsp;&nbsp;</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>ESC</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>&nbsp;&nbsp;</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>*</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>&nbsp;&nbsp;</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">m</font></div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>&nbsp;&nbsp;</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">nL</font></div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>&nbsp;&nbsp;</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">nH</font></div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div>&nbsp;&nbsp;</div>
                                                   </div>
                                                </td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">d1 ... dk</font></div>
                                                   </div>
                                                </td>
                                             </tr>
                                             <tr>
                                                <td style="text-align:left;">
                                                   <div>
                                                      <div>Hex</div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div>1B</div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div>2A</div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">m</font></div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">nL</font></div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">nH</font></div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">d1 ... dk</font></div>
                                                   </div>
                                                </td>
                                             </tr>
                                             <tr>
                                                <td style="text-align:left;">
                                                   <div>
                                                      <div>Decimal</div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div>27</div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div>42</div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">m</font></div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">nL</font></div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">nH</font></div>
                                                   </div>
                                                </td>
                                                <td style=""></td>
                                                <td style="">
                                                   <div>
                                                      <div><font class="parameter">d1 ... dk</font></div>
                                                   </div>
                                                </td>
                                             </tr>
                                          </tbody>
                                       </table>
                                    </div>
                                 </div>
                              </div>   
                              """

class TestCommandFormat(unittest.TestCase):
    
    def test_parse(self):
        from escpos import CommandFormat
        from bs4 import BeautifulSoup

        doc = BeautifulSoup(TEST_ESC_STAR, features="html.parser")
        cmd = CommandFormat.parse(doc)

        self.assertEqual(cmd.ascii, (['ESC', '*'], ['m', 'nL', 'nH', 'd1 ... dk']))
        self.assertEqual(cmd.hex, (['1B', '2A'], ['m', 'nL', 'nH', 'd1 ... dk']))
        self.assertEqual(cmd.decimal, (['27', '42'], ['m', 'nL', 'nH', 'd1 ... dk']))
