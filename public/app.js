var app = angular.module("telescope", ['angular-loading-bar', 'ngAnimate']);

app.controller("meterController", function($scope, $http, $timeout) {
    $scope.stats = {};
    $scope.meters = [];
    $scope.na = true;
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
    var context = cubism.context().serverDelay(6e4).step(6e4).size(500);
    var horizon = context.horizon().height(300).format(d3.format(".2f")).title("Energy").colors(["#bdd7e7","#bae4b3"]);
    var comparison = context.comparison().height(200).formatChange(d3.format(".1f%")).title("Daily Change");
    $scope.setActiveMeter = function(meter) {
        console.log(meter);
        $scope.activeMeter = meter.name;
        var primary = energy($scope.activeMeter), secondary = primary.shift(-24 * 60 * 60 * 1e3);

        d3.select("#chart").call(function(div) {
            div.selectAll(".horizon").selectAll('*').remove();
            div.selectAll(".comparison").selectAll('*').remove();
            div.selectAll(".horizon").call(horizon.metric(primary));
            div.selectAll(".comparison").call(comparison.primary(primary).secondary(secondary));
        });

        context.on("focus", function(i) {
            format = d3.format(".1f");
            d3.selectAll(".horizon .value").style("right", i === null ? null : context.size() - i + "px").text(format(primary.valueAt(Math.floor(i))) + " W");
        });
        updateStats();
    };
    
    d3.select("#chart").call(function(div) {
        div.append("div").attr("class", "axis").call(context.axis().orient("top"));
        div.append("div").attr("class", "rule").call(context.rule());
        div.append("div").attr("class", "horizon");
        div.append("div").attr("class", "comparison");
    });
    function energy(meter) {
        return context.metric(function(start, stop, step, callback) {
            var req = "/" + meter + "/" + start.getTime() / 1e3 + "/" + stop.getTime() / 1e3 + "/" + step / 1e3;
            $http.get(req).success(function(data) {
                console.log(meter, "success");
                if (meter == $scope.activeMeter)
                    $scope.na = false;
                callback(null, data);  
            }).error(function(data) {
                console.log(meter, "error");
                if (meter == $scope.activeMeter)
                    $scope.na = true;
                callback(new Error("unable to load data"));
            });
        });
    }
});