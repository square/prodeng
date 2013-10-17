var data_all;

$.ajax({
    url: "/cgi-bin/dmarc/summary-flot.cgi",
    success: function(data) {
        data_all = data;
    },
    async: false,
    dataType: "json"
});

$(function() {
   $.each(data_all,function(k,v) {
        var e = $("<div class=row><p><strong>" + k + "<strong></p><hr></div>");
        $("#placeholder").append(e);
        $.each(data_all[k],function(from_domain,result) {
            var f = $("<div id='" + from_domain + "'></div>");
            f.append("<p>" + from_domain + "</p>");
            e.append(f);
            $.each(data_all[k][from_domain],function(type,data) {
                var container = $("<div class=span4 />");
                var g = $("<div class='small_plot' id='" + type + "'/>");
                container.append("<p><small><strong>" + type + "</strong></small></p>");
                container.append(g);
                f.append(container);
                $.plot(g,data,{xaxis: { mode: "time", }, series: {stack: 1 }});
            });
        });
    });
});
