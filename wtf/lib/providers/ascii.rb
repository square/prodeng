require 'rubygems'
require 'pp'

module WTF
  class ASCIIProvider < WTF::Provider
    NAME = "ASCII"

    ASCIIMAP = {}
    ASCIIMAP['wtf'] = <<ASCII

             _.-"""""-._
            / .--.....-.\\
           / /          \\\\    MY CPU IS A NEURAL NET PROCESSOR
           ||           ||
           ||  .--.  .--|/
           /`    @  \\ @ |           A LEARNING COMPUTER
           \\_       _)  |
            |    ,____, ;
            | \\   `--' /
         _./\\  '.____.;_
     _.-'  | `\\      |\'-.
   .'       `\ '.   / /   '.
  /           |/ `\\/`\\|     \\

ASCII

    ASCIIMAP['pony'] = <<ASCII

           .,,.
         ,;;*;;;;,
        .-'``;-');;.
       /'  .-.  /*;;
     .'    \d    \;;               .;;;,
    / o      `    \;    ,__.     ,;*;;;*;,
    \__, _.__,'   \_.-') __)--.;;;;;*;;;;,
     `""`;;;\       /-')_) __)  `\' ';;;;;;
        ;*;;;        -') `)_)  |\ |  ;;;;*;
        ;;;;|        `---`    O | | ;;*;;;
        *;*;\|                 O  / ;;;;;*
       ;;;;;/|    .-------\      / ;*;;;;;
      ;;;*;/ \    |        '.   (`. ;;;*;;;
      ;;;;;'. ;   |          )   \ | ;;;;;;
      ,;*;;;;\/   |.        /   /` | ';;;*;
       ;;;;;;/    |/       /   /__/   ';;;
       '*jgs/     |       /    |      ;*;
            `""""`        `""""`     ;'


ASCII

    def initialize
      @output = ''
    end

    def query(thing)
      if ASCIIMAP[thing]
        ASCIIMAP[thing].each_line { |l| newline redize l.chomp }
      end
    end
  end
end
