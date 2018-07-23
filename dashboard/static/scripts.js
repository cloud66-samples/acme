var g;
var chart;

$(document).ready(function() {
    g = new JustGage({
        id: "gauge",
        value: 0,
        min: 0,
        max: 1000,
        title: "Market"
      });
    histogram();
    setInterval(refreshGauge, 1000);
    setInterval(updateHistogram, 1000);
});

function refreshGauge() {
    $.get("/size", function(data) {
        g.refresh(data.size);
    });
}

function updateHistogram() {
    $.ajax({
        url: '/histogram',
        success: function(point) {
            var series = chart.series[0],
                shift = series.data.length > 20;
            // add the point
            chart.series[0].addPoint(point, true, shift);
        },
        cache: false
    });
}

function histogram() {
    chart = Highcharts.stockChart('container', {
        title: {
            text: 'ACME Price Ticker'
        },

        rangeSelector: {
            buttons: [{
                type: 'hour',
                count: 1,
                text: '1h'
            }, {
                type: 'day',
                count: 1,
                text: '1D'
            }, {
                type: 'all',
                count: 1,
                text: 'All'
            }],
            selected: 1,
            inputEnabled: false
        },

        series: [{
            name: 'ACME',
            type: 'candlestick',
            data: [],
            tooltip: {
                valueDecimals: 2
            }
        }]
    });
}
