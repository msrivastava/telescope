var app = angular.module("telescope", []);

app.controller('meterController', function($scope, $http) {
	$scope.meters = [];
	$http.get('/list').success(function(data) {
		$scope.meters = data;
	});
	$scope.activeMeter = '';

	$scope.isActiveMeter = function(meter) {
		return meter.name == $scope.activeMeter;
	};

	$scope.setActiveMeter = function(meter) {
		console.log(meter);
		$scope.activeMeter = meter.name;
		var primary = energy($scope.activeMeter), secondary = primary.shift(-24 * 60 * 60 * 1e3);
		d3.select("#chart").call(function(div) {
			div.selectAll(".horizon").remove();
			div.selectAll(".comparison").remove();
			div.selectAll(".horizon").data([ primary ]).enter().append("div").attr("class", "horizon").call(context.horizon().height(400).format(d3.format(".2f")).title("Energy"));
        	div.selectAll(".comparison").data([ [ primary, secondary ] ]).enter().append("div").attr("class", "comparison").call(context.comparison().height(200).formatChange(d3.format(".1f%")).title("Daily Change"));
		});
	    context.on("focus", function(i) {
	        format = d3.format(".1f");
	        d3.selectAll(".horizon .value").style("right", i === null ? null : context.size() - i + "px").text(format(primary.valueAt(Math.floor(i))) + " W");
	    });
	};
	var context = cubism.context().serverDelay(10 * 1e3).step(10 * 1e3).size(800);


        
    d3.select("#chart").call(function(div) {
        div.append("div").attr("class", "axis").call(context.axis().orient("top"));
        div.append("div").attr("class", "rule").call(context.rule());
    });


    
    function computePower(p) {
        return Math.abs(p.v[9]);
    }
    
    function nonzero(l) {
        var count = 0;
        for (var i = 0; i < l.length; i++) {
            if (l[i] != 0) count++;
        }
        return count;
    }

    function energy(meter) {
        var v = NaN;
        return context.metric(function(start, stop, step, callback) {
            var req = "/" + meter + "/" + start.getTime() / 1e3 + "/" + stop.getTime() / 1e3;
            d3.json(req, function(data) {
                if (!data) return callback(new Error("unable to load data"));
                var values = [];
                var j = 0;
                for (var i = +start; i < +stop; i += step) {
                    while (j < data.length && +Date.parse(data[j].t) < i) {
                        j++;
                    }
                    if (j >= data.length) {
                        values.push(v);
                        continue;
                    }
                    var t = +Date.parse(data[j].t);
                    if (i <= t && t < i + step) {
                        var read = computePower(data[j]);
                        if (read != 0) {
                            v = read
                        }
                    }
                    values.push(v);
                }
                console.log(nonzero(values), data, values);
                callback(null, values);
            });
        });
    }
});