var data_all;

function showReport(id) {
  $.ajax({
    url: "/api/v1/record/report_metadata_id/" + id,
    success: function(data) {
      var e = $("#reportDetailTableBody");
      e.empty();
      $.each(data,function(idx,detail) {
        var row = "<tr>";
        $.each(
           ["header_from","disposition","count",
            "source_ip","dkim","spf"], function(idx,td) {
              row += "<td>" + detail[td] + "</td>";
          });
        e.append(row + "</tr>");
        });
    },
    async: false,
    dataType: "json"
  });
  $("#reportDetailTableDiv").show();
  $("#reportTableDiv").hide();
}

$.ajax({
    url: "/api/v1/reports",
    success: function(data) {
        data_all = data;
    },
    async: false,
    dataType: "json"
});

$(function() {
  $("#reportTableDiv").show();
  $("#reportDetailTableDiv").hide();
  $.each(data_all,function(k,v) {
        var e  = $("#reportsTableBody");
        var tr = e.append("<tr>");
        tr.append('<td><a onclick="showReport(' + v["id"] + ')">' + v["org_name"] + '</a></td>');
        $.each(["begin","end","id"],function(idx,key) {
          tr.append("<td>"  +  v[key]  + "</td>");
        });
   });
});
