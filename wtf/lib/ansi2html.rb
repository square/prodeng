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
      color: black;
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
      color: white;
    }
    .grey {
      color: grey;
    }
  </style>
</head>
<body><pre>
Query: <form action="/wtf"> <input type="text" name="query" value=#{query}> </form><code>
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
        else
          if s.scan(/\e\[0m/) # reset
            @output += (%{</span>})
          else
            @output += (s.scan(/./m))
          end
        end
      end
      if(envelope)
        @output +=  %{</code></pre></body></html>}
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
