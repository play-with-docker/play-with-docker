(function () {
	'use strict';

	var app = angular.module('DockerPlay', ['ngMaterial']);

	app.controller('PlayController', ['$scope', '$log', '$http', '$location', '$timeout', function($scope, $log, $http, $location, $timeout) {
		$scope.sessionId = window.location.pathname.replace('/p/', '');
		$scope.instances = [];
		$scope.selectedInstance = null;

		$scope.newInstance = function() {
			$http({
				method: 'POST',
				url: '/sessions/' + $scope.sessionId + '/instances',
			}).then(function(response) {
				var i = response.data;
				$scope.instances.push(i);
				$scope.showInstance(i);
			}, function(response) {
				console.log('error', response);
			});
		}

		$scope.getSession = function(sessionId) {
			$http({
				method: 'GET',
				url: '/sessions/' + $scope.sessionId,
			}).then(function(response) {
				var i = response.data;
				for (var k in i.instances) {
					var instance = i.instances[k];

					$scope.instances.push(instance);
				}
				if ($scope.instances.length) {
					$scope.showInstance($scope.instances[0]);
				}
			}, function(response) {
				console.log('error', response);
			});
		}

		$scope.showInstance = function(instance) {
			$scope.selectedInstance = instance;
			if (!instance.isAttached) {
				$timeout(function() {createTerminal(instance.name)});
				instance.isAttached = true;
			}
		}

		$scope.deleteInstance = function(instance) {
			$http({
				method: 'DELETE',
				url: '/sessions/' + $scope.sessionId + '/instances/' + instance.name,
			}).then(function(response) {
				$scope.instances = $scope.instances.filter(function(i) { return i.name != instance.name });
				if ($scope.instances.length) {
					$scope.showInstance($scope.instances[0]);
				}
			}, function(response) {
				console.log('error', response);
			});
		}

		$scope.getSession($scope.sessionId);
	}]);
})();
