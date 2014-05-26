var app = angular.module("telescope", ['angular-loading-bar', 'ngAnimate']);

app.controller("meterController", function($scope, $http, $timeout) {
    $scope.stats = {};
    $scope.meters = [];
    var sync = function() {
        console.log("sync");
        $http.get("/list").success(function(data) {
            $scope.meters = data;
            updateStats();
        });
        $timeout(sync, 6e4);
    };
    sync();
    $scope.activeMeter = "";
    $scope.isActiveMeter = function(meter) {
        return meter.name == $scope.activeMeter;
    };
    function updateStats() {
        for (var i in $scope.meters) {
            if ($scope.meters[i].name == $scope.activeMeter) {
                $scope.stats = {};
                for (var j in $scope.meters[i]) {
                    if (j == "name" || j == "addr") {
                        continue
                    }
                    $scope.stats[j] = $scope.meters[i][j];
                }
            }
        }
    }
    $scope.setActiveMeter = function(meter) {
        console.log(meter);
        $scope.activeMeter = meter.name;
        var primary = energy($scope.activeMeter), secondary = primary.shift(-24 * 60 * 60 * 1e3);
        d3.select("#chart").call(function(div) {
            div.selectAll(".horizon").remove();
            div.selectAll(".comparison").remove();
            div.selectAll(".horizon").data([ primary ]).enter().append("div").attr("class", "horizon").call(context.horizon().height(300).format(d3.format(".2f")).title("Energy").colors(["#bdd7e7","#bae4b3"]));
            div.selectAll(".comparison").data([ [ primary, secondary ] ]).enter().append("div").attr("class", "comparison").call(context.comparison().height(200).formatChange(d3.format(".1f%")).title("Daily Change"));
        });
        context.on("focus", function(i) {
            format = d3.format(".1f");
            d3.selectAll(".horizon .value").style("right", i === null ? null : context.size() - i + "px").text(format(primary.valueAt(Math.floor(i))) + " W");
        });
        updateStats();
    };
    var context = cubism.context().serverDelay(6e4).step(6e4).size(500);
    d3.select("#chart").call(function(div) {
        div.append("div").attr("class", "axis").call(context.axis().orient("top"));
        div.append("div").attr("class", "rule").call(context.rule());
    });
    function smooth(data) {
        var kernel = [70, 56, 28, 8, 1];
        for (var i = 0; i < data.length; i++) {
            var c = kernel[0];
            var v = data[i];
            for (var j = 1; j < kernel.length; j++) {
                p = data[i+j];
                m = data[i-j];
                if (p != undefined) {
                    v += p;
                    c += kernel[j];
                }
                if (m != undefined) {
                    v += m;
                    c += kernel[j];
                }
            }
            data[i] = v/c;
        }
        return data;
    }
    function energy(meter) {
        return context.metric(function(start, stop, step, callback) {
            var req = "/" + meter + "/" + start.getTime() / 1e3 + "/" + stop.getTime() / 1e3 + "/" + step / 1e3;
            $http.get(req).success(function(data) {
                if (!data) return callback(new Error("unable to load data"));
                console.log(data.length);
                callback(null, smooth(data));
            });
        });
    }
});