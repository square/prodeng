function Range(host,port) {
    this.host = host;
    this.port = port;
    this.expand = expand;
    this.compress = compress;
}
    
function expand(x) {
    out = [];
    $.ajax({url: "http://" + this.host + ":" + this.port + "/range/list?" + x,
        dataType: 'text',
        success: function(str) {
            var data = str.split("\n");
            for(i in data.sort()) {
                if(data[i].match("^\s*$")) {
                    continue;
                }
                out.push(data[i]);
            }
        },
        error: function(data) {  },
        async: false
   });
   return out;
}

function compress(x) {
    out = "";
    $.ajax({url: "http://" + this.host + ":" + this.port + "/range/expand?" 
                 + x.join(','),
            dataType: 'text',
            success: function(str) {
                out = str;
            },
            error: function(data) {  },
            async: false
    });
        return out;
}
