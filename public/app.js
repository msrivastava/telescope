var app = angular.module("telescope", ['angular-loading-bar', 'ngAnimate']);

app.controller("meterController", function($scope, $http, $timeout) {
    $scope.stats = {};
    $scope.meters = [];
    $scope.anormalyCounter = 0;
    $scope.lastTimestamp = null;
    $scope.na = true;
    $scope.activeMeter = "";
    function sync() {
        console.log("sync");
        $http.get("/list").success(function(data) {
            $scope.meters = data;
            updateStats();
        });
        $timeout(sync, 6e4);
    }
    sync();
    $scope.isActiveMeter = function(meter) {
        return meter.name == $scope.activeMeter;
    };
    function activeMeterStats() {
        for (var i in $scope.meters) {
            if ($scope.meters[i].name == $scope.activeMeter) {
                return {
                     'Avg': $scope.meters[i]['Avg'],
                     'Max':  $scope.meters[i]['Max'],
                     'Min':  $scope.meters[i]['Min'],
                     'Stddev':  $scope.meters[i]['Stddev'],
                };
            }
        }
        return null
    }
    function updateStats() {
        var stats = activeMeterStats();
        if (stats == null) {
            $scope.stats = {};
            return
        }
        $scope.stats = {
            'Avg Power': stats['Avg'].toFixed(2) + ' W',
            'Max Power': stats['Max'].toFixed(2) + ' W',
            'Min Power': stats['Min'].toFixed(2) + ' W',
            'Power Stddev': stats['Stddev'].toFixed(2) + ' W',
            'Energy in last hour': (stats['Avg'] * 3.6).toFixed(2) + ' kJ',
        };
    }
    $scope.anormalyThresh = function() {
        var stats = activeMeterStats();
        if (stats == null || stats['Avg'] == 0) {
            return Number.MAX_VALUE;
        }
        return stats['Avg'] + 3 * stats['Stddev'];
    };
    var context = cubism.context().step(6e4).size(500);
    var horizon = context.horizon().height(300).format(d3.format(".2f")).title("Power (W)").colors(["#bdd7e7","#bae4b3"]);
    var comparison = context.comparison().height(100).formatChange(d3.format(".0f%")).title("Daily Change (%)");
    $scope.setActiveMeter = function(meter) {
        if (meter.name == $scope.activeMeter) {
            return
        }
        console.log(meter);
        $scope.activeMeter = meter.name;
        $scope.anormalyCounter = 0;
        $scope.lastTimestamp = null;
        var primary = energy($scope.activeMeter, true), secondary = energy($scope.activeMeter, false).shift(-24 * 60 * 60 * 1e3);

        d3.select("#chart").call(function(div) {
            div.selectAll(".horizon").call(horizon.remove).call(horizon.metric(primary));
            div.selectAll(".comparison").call(comparison.remove).call(comparison.primary(primary).secondary(secondary));
        });

        context.on("focus", function(i) {
            d3.selectAll(".horizon .value").style("right", i === null ? null : context.size() - i + "px").text(d3.format(".2f")(primary.valueAt(Math.floor(i))));
            d3.selectAll(".comparison .value").style("right", i === null ? null : context.size() - i + "px").text(d3.format(".0f%")(secondary.valueAt(Math.floor(i))));
        });
        updateStats();
    };
    
    d3.select("#chart").call(function(div) {
        div.append("div").attr("class", "axis").call(context.axis().orient("top"));
        div.append("div").attr("class", "rule").call(context.rule());
        div.append("div").attr("class", "horizon");
        div.append("div").attr("class", "comparison");
    });
    function energy(meter, now) {
        return context.metric(function(start, stop, step, callback) {
            var req = "/" + meter + "/" + start.getTime() / 1e3 + "/" + stop.getTime() / 1e3 + "/" + step / 1e3;
            $http.get(req).success(function(data) {
                console.log(meter, "success");
                $scope.na = false;
                if (now && ($scope.lastTimestamp == null || stop > $scope.lastTimestamp)) {
                    var thresh = $scope.anormalyThresh();
                    for (var i in data) {
                        if (data[i] > thresh) {
                            $scope.anormalyCounter++;
                        }
                    }
                    $scope.lastTimestamp = stop;
                }
                callback(null, data);
            }).error(function(data) {
                console.log(meter, "error");
                $scope.na = true;
                callback(new Error("unable to load data"));
            });
        });
    }
});