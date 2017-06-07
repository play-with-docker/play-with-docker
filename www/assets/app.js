(function() {
  'use strict';

  var app = angular.module('DockerPlay', ['ngMaterial']);

  // Automatically redirects user to a new session when bypassing captcha.
  // Controller keeps code/logic separate from the HTML
  app.controller("BypassController", ['$scope', '$log', '$http', '$location', '$timeout', function($scope, $log, $http, $location, $timeout) {
    setTimeout(function() {
      document.getElementById("welcomeFormBypass").submit();
    }, 500);
  }]);

  function SessionBuilderModalController($mdDialog, $scope) {
    $scope.createBuilderTerminal();

    $scope.closeSessionBuilder = function() {
      $mdDialog.cancel();
    }
  }

  app.controller('PlayController', ['$scope', '$log', '$http', '$location', '$timeout', '$mdDialog', '$window', 'TerminalService', 'KeyboardShortcutService', 'InstanceService', 'SessionService', function($scope, $log, $http, $location, $timeout, $mdDialog, $window, TerminalService, KeyboardShortcutService, InstanceService, SessionService) {
    $scope.sessionId = SessionService.getCurrentSessionId();
    $scope.instances = [];
    $scope.idx = {};
    $scope.selectedInstance = null;
    $scope.isAlive = true;
    $scope.ttl = '--:--:--';
    $scope.connected = true;
    $scope.isInstanceBeingCreated = false;
    $scope.newInstanceBtnText = '+ Add new instance';
    $scope.deleteInstanceBtnText = 'Delete';
    $scope.isInstanceBeingDeleted = false;

    var selectedKeyboardShortcuts = KeyboardShortcutService.getCurrentShortcuts();

    angular.element($window).bind('resize', function() {
      if ($scope.selectedInstance) {
        $scope.resize($scope.selectedInstance.term.proposeGeometry());
      }
    });

    $scope.$on("settings:shortcutsSelected", function(e, preset) {
      selectedKeyboardShortcuts = preset;
    });


    $scope.showAlert = function(title, content, parent) {
      $mdDialog.show(
        $mdDialog.alert()
        .parent(angular.element(document.querySelector(parent || '#popupContainer')))
        .clickOutsideToClose(true)
        .title(title)
        .textContent(content)
        .ok('Got it!')
      );
    }

    $scope.resize = function(geometry) {
      $scope.socket.emit('viewport resize', geometry.cols, geometry.rows);
    }

    KeyboardShortcutService.setResizeFunc($scope.resize);

    $scope.closeSession = function() {
      // Remove alert before closing browser tab
      window.onbeforeunload = null;
      $scope.socket.emit('session close');
    }

    $scope.upsertInstance = function(info) {
      var i = info;
      if (!$scope.idx[i.name]) {
        $scope.instances.push(i);
        i.buffer = '';
        $scope.idx[i.name] = i;
      } else {
        $scope.idx[i.name].ip = i.ip;
        $scope.idx[i.name].hostname = i.hostname;
      }

      return $scope.idx[i.name];
    }

    $scope.newInstance = function() {
      updateNewInstanceBtnState(true);
      $http({
        method: 'POST',
        url: '/sessions/' + $scope.sessionId + '/instances',
        data : { ImageName : InstanceService.getDesiredImage() }
      }).then(function(response) {
        var i = $scope.upsertInstance(response.data);
        $scope.showInstance(i);
      }, function(response) {
        if (response.status == 409) {
          $scope.showAlert('Max instances reached', 'Maximum number of instances reached')
        }
      }).finally(function() {
        updateNewInstanceBtnState(false);
      });
    }

    $scope.setSessionState = function(state) {
      $scope.ready = state;

      if (!state) {
        $mdDialog.show({
          controller: SessionBuilderModalController,
          contentElement: '#builderDialog',
          parent: angular.element(document.body),
          clickOutsideToClose: false,
          scope: $scope,
          preserveScope: true
        });
      }
    }

    $scope.getSession = function(sessionId) {
      $http({
        method: 'GET',
        url: '/sessions/' + $scope.sessionId,
      }).then(function(response) {
        $scope.setSessionState(response.data.ready);

        if (response.data.created_at) {
          $scope.expiresAt = moment(response.data.expires_at);
          setInterval(function() {
            $scope.ttl = moment.utc($scope.expiresAt.diff(moment())).format('HH:mm:ss');
            $scope.$apply();
          }, 1000);
        }
        var socket = io({ path: '/sessions/' + sessionId + '/ws' });

        socket.on('session ready', function(ready) {
          $scope.setSessionState(ready);
        });

        socket.on('session builder out', function(data) {
          $scope.builderTerminal.write(data);
        });

        socket.on('terminal out', function(name, data) {
          var instance = $scope.idx[name];

          if (!instance) {
            // instance is new and was created from another client, we should add it
            $scope.upsertInstance({ name: name });
            instance = $scope.idx[name];
          }
          if (!instance.term) {
            instance.buffer += data;
          } else {
            instance.term.write(data);
          }
        });

        socket.on('session end', function() {
          $scope.showAlert('Session timed out!', 'Your session has expired and all of your instances have been deleted.', '#sessionEnd')
          $scope.isAlive = false;
        });

        socket.on('viewport', function(rows, cols) {
        });

        socket.on('new instance', function(name, ip, hostname) {
          $scope.upsertInstance({ name: name, ip: ip, hostname: hostname });
          $scope.$apply(function() {
            if ($scope.instances.length == 1) {
              $scope.showInstance($scope.instances[0]);
            }
          });
        });

        socket.on('delete instance', function(name) {
          $scope.removeInstance(name);
          $scope.$apply();
        });

        socket.on('viewport resize', function(cols, rows) {
          // viewport has changed, we need to resize all terminals

          $scope.instances.forEach(function(instance) {
            instance.term.resize(cols, rows);
          });
        });

        socket.on('connect_error', function() {
          $scope.connected = false;
        });
        socket.on('connect', function() {
          $scope.connected = true;
        });

        socket.on('instance stats', function(name, mem, cpu, isManager, ports) {
          $scope.idx[name].mem = mem;
          $scope.idx[name].cpu = cpu;
          $scope.idx[name].isManager = isManager;
          $scope.idx[name].ports = ports;
          $scope.$apply();
        });

        $scope.socket = socket;

        var i = response.data;
        for (var k in i.instances) {
          var instance = i.instances[k];
          $scope.instances.push(instance);
          $scope.idx[instance.name] = instance;
        }

        // If instance is passed in URL, select it
        let inst = $scope.idx[$location.hash()];
        if (inst) $scope.showInstance(inst);
      }, function(response) {
        if (response.status == 404) {
          document.write('session not found');
          return
        }
      });
    }

    $scope.getProxyUrl = function(instance, port) {
      var url = window.location.protocol + '//pwd' + instance.ip.replace(/\./g, '-') + '-' + port + '.' + window.location.host;

      return url;
    }

    $scope.showInstance = function(instance) {
      $scope.selectedInstance = instance;
      $location.hash(instance.name);
      if (!instance.creatingTerminal) {
        if (!instance.term) {
          $timeout(function() {
            createTerminal(instance);
            TerminalService.setFontSize(TerminalService.getFontSize());
            instance.term.focus();
          }, 0, false);
          return
        }
      }
      $timeout(function() {
        instance.term.focus();
      }, 0, false);
    }

    $scope.removeInstance = function(name) {
      if ($scope.idx[name]) {
        delete $scope.idx[name];
        $scope.instances = $scope.instances.filter(function(i) {
          return i.name != name;
        });
        if ($scope.instances.length) {
          $scope.showInstance($scope.instances[0]);
        }
      }
    }

    $scope.deleteInstance = function(instance) {
      updateDeleteInstanceBtnState(true);
      $http({
        method: 'DELETE',
        url: '/sessions/' + $scope.sessionId + '/instances/' + instance.name,
      }).then(function(response) {
        $scope.removeInstance(instance.name);
      }, function(response) {
        console.log('error', response);
      }).finally(function() {
        updateDeleteInstanceBtnState(false);
      });
    }

    $scope.getSession($scope.sessionId);

    $scope.createBuilderTerminal = function() {
      var builderTerminalContainer = document.getElementById('builder-terminal');
      // For some reason the dialog DOM might not be ready, so we just keep trying
      if (!builderTerminalContainer) {
        setTimeout($scope.createBuilderTerminal, 100);
        return;
      }
      let term = new Terminal({
        cursorBlink: false
      });

      term.open(builderTerminalContainer);
      $scope.builderTerminal = term;
    }
    function createTerminal(instance, cb) {
      if (instance.term) {
        return instance.term;
      }

      var terminalContainer = document.getElementById('terminal-' + instance.name);

      var term = new Terminal({
        cursorBlink: false
      });

      term.attachCustomKeydownHandler(function(e) {
        // Ctrl + Alt + C
        if (e.ctrlKey && e.altKey && (e.keyCode == 67)) {
          document.execCommand('copy');
          return false;
        }
      });

      term.attachCustomKeydownHandler(function(e) {
        if (selectedKeyboardShortcuts == null)
          return;
        var presets = selectedKeyboardShortcuts.presets
        .filter(function(preset) { return preset.keyCode == e.keyCode })
        .filter(function(preset) { return (preset.metaKey == undefined && !e.metaKey) || preset.metaKey == e.metaKey })
        .filter(function(preset) { return (preset.ctrlKey == undefined && !e.ctrlKey) || preset.ctrlKey == e.ctrlKey })
        .filter(function(preset) { return (preset.altKey == undefined && !e.altKey) || preset.altKey == e.altKey })
        .forEach(function(preset) { preset.action({ terminal : term })});
      });

      term.open(terminalContainer);

      // Set geometry during the next tick, to avoid race conditions.
      setTimeout(function() {
        $scope.resize(term.proposeGeometry());
      }, 4);

      term.on('data', function(d) {
        $scope.socket.emit('terminal in', instance.name, d);
      });

      instance.term = term;

      if (instance.buffer) {
        term.write(instance.buffer);
        instance.buffer = '';
      }

      if (cb) {
        cb();
      }
    }

    function updateNewInstanceBtnState(isInstanceBeingCreated) {
      if (isInstanceBeingCreated === true) {
        $scope.newInstanceBtnText = '+ Creating...';
        $scope.isInstanceBeingCreated = true;
      } else {
        $scope.newInstanceBtnText = '+ Add new instance';
        $scope.isInstanceBeingCreated = false;
      }
    }

    function updateDeleteInstanceBtnState(isInstanceBeingDeleted) {
      if (isInstanceBeingDeleted === true) {
        $scope.deleteInstanceBtnText = 'Deleting...';
        $scope.isInstanceBeingDeleted = true;
      } else {
        $scope.deleteInstanceBtnText = 'Delete';
        $scope.isInstanceBeingDeleted = false;
      }
    }
  }])
  .config(['$mdIconProvider', '$locationProvider', function($mdIconProvider, $locationProvider) {
    $locationProvider.html5Mode({enabled: true, requireBase: false});
    $mdIconProvider.defaultIconSet('../assets/social-icons.svg', 24);
  }])
  .component('settingsIcon', {
    template : "<md-button class='md-mini' ng-click='$ctrl.onClick()'><md-icon class='material-icons'>settings</md-icon></md-button>",
    controller : function($mdDialog) {
      var $ctrl = this;
      $ctrl.onClick = function() {
        $mdDialog.show({
          controller : function() {},
          template : "<settings-dialog></settings-dialog>",
          parent: angular.element(document.body),
          clickOutsideToClose : true
        })
      }
    }
  })
  .component('templatesIcon', {
    template : "<md-button class='md-mini' ng-click='$ctrl.onClick()'><md-icon class='material-icons'>build</md-icon></md-button>",
    controller : function($mdDialog) {
      var $ctrl = this;
      $ctrl.onClick = function() {
        $mdDialog.show({
          controller : function() {},
          template : "<templates-dialog></templates-dialog>",
          parent: angular.element(document.body),
          clickOutsideToClose : true
        })
      }
    }
  })
  .component("templatesDialog", {
    templateUrl : "templates-modal.html",
    controller : function($mdDialog, $scope, SessionService) {
      var $ctrl = this;
      $scope.building = false;
	  $scope.templates = SessionService.getAvailableTemplates();
      $ctrl.close = function() {
        $mdDialog.cancel();
      }
	  $ctrl.setupSession = function(setup) {
		$scope.building = true;
		SessionService.setup(setup, function(err) {
            $scope.building = false;
			if (err) {
				$scope.errorMessage = err;
				return;
			}
			$ctrl.close();
		});
	  }
    }
  })
  .component("settingsDialog", {
    templateUrl : "settings-modal.html",
    controller : function($mdDialog, KeyboardShortcutService, $rootScope, InstanceService, TerminalService) {
      var $ctrl = this;
      $ctrl.$onInit = function() {
        $ctrl.keyboardShortcutPresets = KeyboardShortcutService.getAvailablePresets();
        $ctrl.selectedShortcutPreset = KeyboardShortcutService.getCurrentShortcuts();
        $ctrl.instanceImages = InstanceService.getAvailableImages();
        $ctrl.selectedInstanceImage = InstanceService.getDesiredImage();
        $ctrl.terminalFontSizes = TerminalService.getFontSizes();
      };

      $ctrl.currentShortcutConfig = function(value) {
        if (value !== undefined) {
          value = JSON.parse(value);
          KeyboardShortcutService.setCurrentShortcuts(value);
          $ctrl.selectedShortcutPreset = angular.copy(KeyboardShortcutService.getCurrentShortcuts());
          $rootScope.$broadcast('settings:shortcutsSelected', $ctrl.selectedShortcutPreset);
        }
        return JSON.stringify(KeyboardShortcutService.getCurrentShortcuts());
      };

      $ctrl.currentDesiredInstanceImage = function(value) {
        if (value !== undefined) {
          InstanceService.setDesiredImage(value);
        }
        return InstanceService.getDesiredImage(value);
      };
      $ctrl.currentTerminalFontSize = function(value) {
        if (value !== undefined) {
          // set font size
          TerminalService.setFontSize(value);
          return;
        }

        return TerminalService.getFontSize();
      }

      $ctrl.close = function() {
        $mdDialog.cancel();
      }
    }
  })
  .service("SessionService", function($http) {
	var templates = [
		{
			title: '3 Managers and 2 Workers',
			icon: '/assets/swarm.png',
			setup: {
				instances: [
					{hostname: 'manager1', is_swarm_manager: true},
					{hostname: 'manager2', is_swarm_manager: true},
					{hostname: 'manager3', is_swarm_manager: true},
					{hostname: 'worker1', is_swarm_worker: true},
					{hostname: 'worker2', is_swarm_worker: true}
				]
			}
		},
		{
			title: '5 Managers and no workers',
			icon: '/assets/swarm.png',
			setup: {
				instances: [
					{hostname: 'manager1', is_swarm_manager: true},
					{hostname: 'manager2', is_swarm_manager: true},
					{hostname: 'manager3', is_swarm_manager: true},
					{hostname: 'manager4', is_swarm_manager: true},
					{hostname: 'manager5', is_swarm_manager: true}
				]
			}
		}
	];

    return {
      getAvailableTemplates: getAvailableTemplates,
	  getCurrentSessionId: getCurrentSessionId,
	  setup: setup,
    };

	function getCurrentSessionId() {
	  return window.location.pathname.replace('/p/', '');
    }
    function getAvailableTemplates() {
      return templates;
    }
    function setup(plan, cb) {
      return $http
      .post("/sessions/" + getCurrentSessionId() + "/setup", plan)
      .then(function(response) {
		if (cb) cb();
      }, function(response) {
		if (cb) cb(response.data);
	  });
    }
  })
  .service("InstanceService", function($http) {
    var instanceImages = [];
    _prepopulateAvailableImages();

    return {
      getAvailableImages : getAvailableImages,
      setDesiredImage : setDesiredImage,
      getDesiredImage : getDesiredImage,
    };

    function getAvailableImages() {
      return instanceImages;
    }

    function getDesiredImage() {
      var image = localStorage.getItem("settings.desiredImage");
      if (image == null)
        return instanceImages[0];
      return image;
    }

    function setDesiredImage(image) {
      if (image === null)
        localStorage.removeItem("settings.desiredImage");
      else
        localStorage.setItem("settings.desiredImage", image);
    }

    function _prepopulateAvailableImages() {
      return $http
      .get("/instances/images")
      .then(function(response) {
        instanceImages = response.data;
      });
    }

  })
  .run(function(InstanceService) { /* forcing pre-populating for now */ })
  .service("KeyboardShortcutService", ['TerminalService', function(TerminalService) {
    var resizeFunc;

    return {
      getAvailablePresets : getAvailablePresets,
      getCurrentShortcuts : getCurrentShortcuts,
      setCurrentShortcuts : setCurrentShortcuts,
      setResizeFunc : setResizeFunc
    };

    function setResizeFunc(f) {
      resizeFunc = f;
    }

    function getAvailablePresets() {
      return [
        { name : "None", presets : [
          { description : "Toggle terminal fullscreen", command : "Alt+enter", altKey : true, keyCode : 13, action : function(context) { TerminalService.toggleFullscreen(context.terminal, resizeFunc); }}
        ] },
        {
          name : "Mac OSX",
          presets : [
            { description : "Clear terminal", command : "Cmd+K", metaKey : true, keyCode : 75, action : function(context) { context.terminal.clear(); }},
            { description : "Toggle terminal fullscreen", command : "Alt+enter", altKey : true, keyCode : 13, action : function(context) { TerminalService.toggleFullscreen(context.terminal, resizeFunc); }}
          ]
        }
      ]
    }

    function getCurrentShortcuts() {
      var shortcuts = localStorage.getItem("shortcut-preset-name");
      if (shortcuts == null) {
        shortcuts = getDefaultShortcutPrefixName();
        if (shortcuts == null)
          return null;
      }

      var preset = getAvailablePresets()
      .filter(function(preset) { return preset.name == shortcuts; });
      if (preset.length == 0)
        console.error("Unable to find preset with name '" + shortcuts + "'");
      return preset[0];
      return (shortcuts == null) ? null : JSON.parse(shortcuts);
    }

    function setCurrentShortcuts(config) {
      localStorage.setItem("shortcut-preset-name", config.name);
    }

    function getDefaultShortcutPrefixName() {
      if (window.navigator.platform.toUpperCase().indexOf('MAC') >= 0)
        return "Mac OSX";
      return "None";
    }
  }])
  .service('TerminalService', ['$window', function($window) {
    var fullscreen;
    var fontSize = getFontSize();
    return {
      getFontSizes : getFontSizes,
      setFontSize : setFontSize,
      getFontSize : getFontSize,
      increaseFontSize : increaseFontSize,
      decreaseFontSize : decreaseFontSize,
      toggleFullscreen : toggleFullscreen
    };
    function getFontSizes() {
      var terminalFontSizes = [];
      for (var i=3; i<40; i++) {
        terminalFontSizes.push(i+'px');
      }
      return terminalFontSizes;
    };
    function getFontSize() {
      if (!fontSize) {
        return $('.terminal').css('font-size');
      }
      return fontSize;
    }
    function setFontSize(value) {
      fontSize = value;
      var size = parseInt(value);
      $('.terminal').css('font-size', value).css('line-height', (size + 2)+'px');
      //.css('line-height', value).css('height', value);
      angular.element($window).trigger('resize');
    }
    function increaseFontSize() {
      var sizes = getFontSizes();
      var size = getFontSize();
      var i = sizes.indexOf(size);
      if (i == -1) {
        return;
      }
      if (i+1 > sizes.length) {
        return;
      }
      setFontSize(sizes[i+1]);
    }
    function decreaseFontSize() {
      var sizes = getFontSizes();
      var size = getFontSize();
      var i = sizes.indexOf(size);
      if (i == -1) {
        return;
      }
      if (i-1 < 0) {
        return;
      }
      setFontSize(sizes[i-1]);
    }
    function toggleFullscreen(terminal, resize) {
      if(fullscreen) {
        terminal.toggleFullscreen();
        resize(fullscreen);
        fullscreen = null;
      } else {
        fullscreen = terminal.proposeGeometry();
        terminal.toggleFullscreen();
        angular.element($window).trigger('resize');
      }
    }
  }]);
})();
