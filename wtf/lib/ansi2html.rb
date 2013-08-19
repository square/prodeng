require 'strscan'

module WTF
  class Ansi2html
    COLOR = {
       '1' => 'bold',
      '30' => 'black',
      '31' => 'red',
      '32' => 'green',
      '33' => 'yellow',
      '34' => 'blue',
      '35' => 'magenta',
      '36' => 'cyan',
      '37' => 'white',
      '90' => 'grey'
    }

    XTERM_COLORS = {
      # xterm-16 colors
      "000"=>"000000", "001"=>"800000", "002"=>"008000", "003"=>"808000", "004"=>"000080", "005"=>"800080",
      "006"=>"008080", "007"=>"c0c0c0", "008"=>"808080", "009"=>"ff0000", "010"=>"00ff00", "011"=>"ffff00",
      "012"=>"0000ff", "013"=>"ff00ff", "014"=>"00ffff", "015"=>"ffffff",
      # xterm-256 colors
      "016"=>"000000", "017"=>"00005f", "018"=>"000087", "019"=>"0000af", "020"=>"0000d7", "021"=>"0000ff",
      "022"=>"005f00", "023"=>"005f5f", "024"=>"005f87", "025"=>"005faf", "026"=>"005fd7", "027"=>"005fff",
      "028"=>"008700", "029"=>"00875f", "030"=>"008787", "031"=>"0087af", "032"=>"0087d7", "033"=>"0087ff",
      "034"=>"00af00", "035"=>"00af5f", "036"=>"00af87", "037"=>"00afaf", "038"=>"00afd7", "039"=>"00afff",
      "040"=>"00d700", "041"=>"00d75f", "042"=>"00d787", "043"=>"00d7af", "044"=>"00d7d7", "045"=>"00d7ff",
      "046"=>"00ff00", "047"=>"00ff5f", "048"=>"00ff87", "049"=>"00ffaf", "050"=>"00ffd7", "051"=>"00ffff",
      "052"=>"5f0000", "053"=>"5f005f", "054"=>"5f0087", "055"=>"5f00af", "056"=>"5f00d7", "057"=>"5f00ff",
      "058"=>"5f5f00", "059"=>"5f5f5f", "060"=>"5f5f87", "061"=>"5f5faf", "062"=>"5f5fd7", "063"=>"5f5fff",
      "064"=>"5f8700", "065"=>"5f875f", "066"=>"5f8787", "067"=>"5f87af", "068"=>"5f87d7", "069"=>"5f87ff",
      "070"=>"5faf00", "071"=>"5faf5f", "072"=>"5faf87", "073"=>"5fafaf", "074"=>"5fafd7", "075"=>"5fafff",
      "076"=>"5fd700", "077"=>"5fd75f", "078"=>"5fd787", "079"=>"5fd7af", "080"=>"5fd7d7", "081"=>"5fd7ff",
      "082"=>"5fff00", "083"=>"5fff5f", "084"=>"5fff87", "085"=>"5fffaf", "086"=>"5fffd7", "087"=>"5fffff",
      "088"=>"870000", "089"=>"87005f", "090"=>"870087", "091"=>"8700af", "092"=>"8700d7", "093"=>"8700ff",
      "094"=>"875f00", "095"=>"875f5f", "096"=>"875f87", "097"=>"875faf", "098"=>"875fd7", "099"=>"875fff",
      "100"=>"878700", "101"=>"87875f", "102"=>"878787", "103"=>"8787af", "104"=>"8787d7", "105"=>"8787ff",
      "106"=>"87af00", "107"=>"87af5f", "108"=>"87af87", "109"=>"87afaf", "110"=>"87afd7", "111"=>"87afff",
      "112"=>"87d700", "113"=>"87d75f", "114"=>"87d787", "115"=>"87d7af", "116"=>"87d7d7", "117"=>"87d7ff",
      "118"=>"87ff00", "119"=>"87ff5f", "120"=>"87ff87", "121"=>"87ffaf", "122"=>"87ffd7", "123"=>"87ffff",
      "124"=>"af0000", "125"=>"af005f", "126"=>"af0087", "127"=>"af00af", "128"=>"af00d7", "129"=>"af00ff",
      "130"=>"af5f00", "131"=>"af5f5f", "132"=>"af5f87", "133"=>"af5faf", "134"=>"af5fd7", "135"=>"af5fff",
      "136"=>"af8700", "137"=>"af875f", "138"=>"af8787", "139"=>"af87af", "140"=>"af87d7", "141"=>"af87ff",
      "142"=>"afaf00", "143"=>"afaf5f", "144"=>"afaf87", "145"=>"afafaf", "146"=>"afafd7", "147"=>"afafff",
      "148"=>"afd700", "149"=>"afd75f", "150"=>"afd787", "151"=>"afd7af", "152"=>"afd7d7", "153"=>"afd7ff",
      "154"=>"afff00", "155"=>"afff5f", "156"=>"afff87", "157"=>"afffaf", "158"=>"afffd7", "159"=>"afffff",
      "160"=>"d70000", "161"=>"d7005f", "162"=>"d70087", "163"=>"d700af", "164"=>"d700d7", "165"=>"d700ff",
      "166"=>"d75f00", "167"=>"d75f5f", "168"=>"d75f87", "169"=>"d75faf", "170"=>"d75fd7", "171"=>"d75fff",
      "172"=>"d78700", "173"=>"d7875f", "174"=>"d78787", "175"=>"d787af", "176"=>"d787d7", "177"=>"d787ff",
      "178"=>"dfaf00", "179"=>"dfaf5f", "180"=>"dfaf87", "181"=>"dfafaf", "182"=>"dfafdf", "183"=>"dfafff",
      "184"=>"dfdf00", "185"=>"dfdf5f", "186"=>"dfdf87", "187"=>"dfdfaf", "188"=>"dfdfdf", "189"=>"dfdfff",
      "190"=>"dfff00", "191"=>"dfff5f", "192"=>"dfff87", "193"=>"dfffaf", "194"=>"dfffdf", "195"=>"dfffff",
      "196"=>"ff0000", "197"=>"ff005f", "198"=>"ff0087", "199"=>"ff00af", "200"=>"ff00df", "201"=>"ff00ff",
      "202"=>"ff5f00", "203"=>"ff5f5f", "204"=>"ff5f87", "205"=>"ff5faf", "206"=>"ff5fdf", "207"=>"ff5fff",
      "208"=>"ff8700", "209"=>"ff875f", "210"=>"ff8787", "211"=>"ff87af", "212"=>"ff87df", "213"=>"ff87ff",
      "214"=>"ffaf00", "215"=>"ffaf5f", "216"=>"ffaf87", "217"=>"ffafaf", "218"=>"ffafdf", "219"=>"ffafff",
      "220"=>"ffdf00", "221"=>"ffdf5f", "222"=>"ffdf87", "223"=>"ffdfaf", "224"=>"ffdfdf", "225"=>"ffdfff",
      "226"=>"ffff00", "227"=>"ffff5f", "228"=>"ffff87", "229"=>"ffffaf", "230"=>"ffffdf", "231"=>"ffffff",
      # xterm-greyscale colors
      "232"=>"080808", "233"=>"121212", "234"=>"1c1c1c", "235"=>"262626", "236"=>"303030", "237"=>"3a3a3a",
      "238"=>"444444", "239"=>"4e4e4e", "240"=>"585858", "241"=>"626262", "242"=>"6c6c6c", "243"=>"767676",
      "244"=>"808080", "245"=>"8a8a8a", "246"=>"949494", "247"=>"9e9e9e", "248"=>"a8a8a8", "249"=>"b2b2b2", 
      "250"=>"bcbcbc", "251"=>"c6c6c6", "252"=>"d0d0d0", "253"=>"dadada", "254"=>"e4e4e4", "255"=>"eeeeee",
    }

    attr_accessor :output
    def initialize(ansi, query=nil, envelope=true, black=true)
      @output = ""
      if(envelope)
        background, color = black ? %w(black white) : %w(white black)
        @output +=  %{<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    body {
      background-color: #{background}; color: #{color}; font-size: 16;
    }
    code {
      font-size: 150%;
    }

    .bold {
      font-weight: bold;
    }
    .black {
      color: #333;
    }
    .red {
      color: red;
    }
    .green {
      color: green;
    }
    .yellow {
      color: yellow;
    }
    .blue {
      color: blue;
    }
    .magenta {
      color: magenta;
    }
    .cyan {
      color: cyan;
    }
    .white {
      color: #eee;
    }
    .grey {
      color: grey;
    }
  </style>
</head>
<body ><pre>
Query: <form action="/wtf"> <input type="text" id="query" name="query" value=#{query}> </form><code>
<br>
}
      end

      s = StringScanner.new(ansi.gsub("<", "&lt;"))

      while(!s.eos?)
        if s.scan(/\e\[(3[0-7]|90|1)m/) # color
          @output += (%{<span class="#{COLOR[s[1]]}">})

          if COLOR[s[1]] == 'cyan' # scan for urls
            output, next_thing = scan_until_colorchange(s)
            @output += "<a style=\"color:cyan;\" href='#{output}'>#{output}</a>"
            @output += next_thing
          end
          if COLOR[s[1]] == 'yellow' # scan for wtflinks
            output, next_thing = scan_until_colorchange(s)
            @output += "<a style=\"color:yellow;\" href='?query=#{output}'>#{output}</a>"
            @output += next_thing
          end
        elsif s.scan(/\e\[48\;5\;(\d+)m /) # xterm256 bg color (ascii photo pixel)
          colorspec = "#" + XTERM_COLORS["%03d" % s[1].to_i]
          @output += (%{<span style="background-color:#{colorspec};">&nbsp;</span>})
          
        else
          if s.scan(/\e\[0m/) # reset
            @output += (%{</span>})
          else
            @output += (s.scan(/./m))
          end
        end
      end
      if(envelope)
        @output +=  %{</code></pre><script type="text/javascript">document.getElementById('query').focus()</script></body></html>}
      end

    end

    def scan_until_colorchange(s)
      running_output = ''
      while(!s.eos?)
        if s.scan(/\e\[(3[0-7]|90|1)m/) # color
          return [running_output, %{<span class="#{COLOR[s[1]]}">}]
        elsif s.scan(/\e\[0m/) # reset
          return [running_output, %{</span>}]
        else
          running_output += s.scan(/./m)
        end
      end
      return [running_output, '']
    end


  end
end
