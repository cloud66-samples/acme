var g;
$(document).ready(function() {
    g = new JustGage({
        id: "gauge",
        value: 0,
        min: 0,
        max: 1000,
        title: "Market"
      });
    var tid = setInterval(refreshGauge, 1000);
});

function refreshGauge() {
    $.get("/size", function(data) {
        g.refresh(data.size);
    });
}
