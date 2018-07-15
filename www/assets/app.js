(function() {
  'use strict';

  var app = angular.module('DockerPlay', ['ngMaterial', 'ngFileUpload', 'ngclipboard']);

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

  app.controller('PlayController', ['$scope', '$log', '$http', '$location', '$timeout', '$mdDialog', '$window', 'TerminalService', 'KeyboardShortcutService', 'InstanceService', 'SessionService', 'Upload', function($scope, $log, $http, $location, $timeout, $mdDialog, $window, TerminalService, KeyboardShortcutService, InstanceService, SessionService, Upload) {
    $scope.sessionId = SessionService.getCurrentSessionId();
    $scope.instances = [];
    $scope.idx = {};
    $scope.host = window.location.host;
    $scope.idxByHostname = {};
    $scope.selectedInstance = null;
    $scope.isAlive = true;
    $scope.ttl = '--:--:--';
    $scope.connected = false;
    $scope.type = {windows: false};
    $scope.isInstanceBeingCreated = false;
    $scope.newInstanceBtnText = '+ Add new instance';
    $scope.deleteInstanceBtnText = 'Delete';
    $scope.isInstanceBeingDeleted = false;
    $scope.uploadProgress = 0;

    $scope.uploadFiles = function (files, invalidFiles) {
        let total = files.length;
        let uploadFile = function() {
          let file = files.shift();
          if (!file){
            $scope.uploadMessage = "";
            $scope.uploadProgress = 0;
            return
          }
          $scope.uploadMessage = "Uploading file(s) " + (total - files.length) + "/"+ total + " : " + file.name;
          let upload = Upload.upload({url: '/sessions/' + $scope.sessionId + '/instances/' + $scope.selectedInstance.name + '/uploads', data: {file: file}, method: 'POST'})
            .then(function(){}, function(){}, function(evt) {
              $scope.uploadProgress = parseInt(100.0 * evt.loaded / evt.total);
            });

          // process next file
          upload.finally(uploadFile);
        }

        uploadFile();
    }

    var selectedKeyboardShortcuts = KeyboardShortcutService.getCurrentShortcuts();

    $scope.resizeHandler = null;

    angular.element($window).bind('resize', function() {
      if ($scope.selectedInstance) {
        if (!$scope.resizeHandler) {
            $scope.resizeHandler = setTimeout(function() {
                $scope.resizeHandler = null
                $scope.resize($scope.selectedInstance.term.proposeGeometry());
            }, 1000);
        }
      }
    });

    $scope.$on("settings:shortcutsSelected", function(e, preset) {
      selectedKeyboardShortcuts = preset;
    });


    $scope.showAlert = function(title, content, parent, cb) {
      $mdDialog.show(
        $mdDialog.alert()
        .parent(angular.element(document.querySelector(parent || '#popupContainer')))
        .clickOutsideToClose(true)
        .title(title)
        .textContent(content)
        .ok('Got it!')
      ).finally(function() {
        if (cb) {
           cb();
        }
      });
    }

    $scope.resize = function(geometry) {
      $scope.socket.emit('instance viewport resize', geometry.cols, geometry.rows);
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
        $scope.idxByHostname[i.hostname] = i;
      } else {
        $scope.idx[i.name] = Object.assign($scope.idx[i.name], info);
      }

      return $scope.idx[i.name];
    }

    $scope.newInstance = function() {
      updateNewInstanceBtnState(true);
      var instanceType = $scope.type.windows ? 'windows': 'linux';
      $http({
        method: 'POST',
        url: '/sessions/' + $scope.sessionId + '/instances',
        data : { ImageName : InstanceService.getDesiredImage(), type: instanceType }
      }).then(function(response) {
        $scope.upsertInstance(response.data);
      }, function(response) {
        if (response.status == 409) {
          $scope.showAlert('Max instances reached', 'Maximum number of instances reached')
        } else if (response.status == 503 && response.data.error == 'out_of_capacity') {
          $scope.showAlert('Out Of Capacity', 'We are really sorry. But we are currently out of capacity and cannot create new instances. Please try again later.')
        }
      }).finally(function() {
        updateNewInstanceBtnState(false);
      });
    }

    $scope.setSessionState = function(state) {
      $scope.ready = state;

      if (!state) {
        $mdDialog.show({
          onComplete: function(){SessionBuilderModalController($mdDialog, $scope)},
          contentElement: '#builderDialog',
          parent: angular.element(document.body),
          clickOutsideToClose: false,
          scope: $scope,
          preserveScope: true
        });
      }
    }

    $scope.loadPlaygroundConf = function() {
      $http({
        method: 'GET',
        url: '/my/playground',
      }).then(function(response) {
        $scope.playground = response.data;
      });

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

        var i = response.data;
        for (var k in i.instances) {
          var instance = i.instances[k];
          $scope.instances.push(instance);
          $scope.idx[instance.name] = instance;
          $scope.idxByHostname[instance.hostname] = instance;
        }

	var base = '';
	if (window.location.protocol == 'http:') {
		base = 'ws://';
	} else {
		base = 'wss://';
	}
	base += window.location.host;
	if (window.location.port) {
		base += ':' + window.location.port;
	}

	var socket = new ReconnectingWebSocket(base + '/sessions/' + sessionId + '/ws/', null, {reconnectInterval: 1000});
	socket.listeners = {};

	socket.on = function(name, cb) {
		if (!socket.listeners[name]) {
			socket.listeners[name] = [];
		}
		socket.listeners[name].push(cb);
	}

	socket.emit = function() {
		var name = arguments[0]
		var args = [];
		for (var i = 1; i < arguments.length; i++) {
			args.push(arguments[i]);
		}
		socket.send(JSON.stringify({name: name, args: args}));
	}

	socket.addEventListener('open', function (event) {
          $scope.connected = true;
	  for (var i in $scope.instances) {
		  var instance = $scope.instances[i];
		  if (instance.term) {
			  instance.term.setOption('disableStdin', false);
		  }
	  }
	});
	socket.addEventListener('close', function (event) {
          $scope.connected = false;
	  for (var i in $scope.instances) {
		  var instance = $scope.instances[i];
		  if (instance.term) {
			  instance.term.setOption('disableStdin', true);
		  }
	  }
	});
	socket.addEventListener('message', function (event) {
		var m = JSON.parse(event.data);
		var ls = socket.listeners[m.name];
		if (ls) {
			for (var i=0; i<ls.length; i++) {
				var l = ls[i];
				l.apply(l, m.args);
			}
		}
	});


        socket.on('instance terminal status', function(name, status) {
            var instance = $scope.idx[name];
            if (instance) {
                instance.status = status;
            }
        });

        socket.on('session ready', function(ready) {
          $scope.setSessionState(ready);
        });

        socket.on('session builder out', function(data) {
          $scope.builderTerminal.write(data);
        });

        socket.on('instance terminal out', function(name, data) {
          var instance = $scope.idx[name];
          if (!instance) {
            return;
          }

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
          $scope.showAlert('Session timed out!', 'Your session has expired and all of your instances have been deleted.', '#sessionEnd', function() {
            window.location.href = '/';
          });
          $scope.isAlive = false;
          socket.close();
        });

        socket.on('instance new', function(name, ip, hostname, proxyHost) {
          var instance = $scope.upsertInstance({ name: name, ip: ip, hostname: hostname, proxy_host: proxyHost, session_id: $scope.sessionId});
          $scope.$apply(function() {
            $scope.showInstance(instance);
          });
        });

        socket.on('instance delete', function(name) {
          $scope.removeInstance(name);
          $scope.$apply();
        });

        socket.on('instance viewport resize', function(cols, rows) {
            if (cols == 0 || rows == 0) {
                return
            }
          // viewport has changed, we need to resize all terminals
          $scope.instances.forEach(function(instance) {
              if (instance.term) {
                instance.term.resize(cols, rows);
                if (instance.buffer) {
                  instance.term.write(instance.buffer);
                  instance.buffer = '';
                }
              }
          });
        });

        socket.on('instance stats', function(stats) {
          if (! $scope.idx[stats.instance]) {
              return
          }
          $scope.idx[stats.instance].mem = stats.mem;
          $scope.idx[stats.instance].cpu = stats.cpu;
          $scope.$apply();
        });

        socket.on('instance docker swarm status', function(status) {
            if (!$scope.idx[status.instance]) {
                return
            }
            if (status.is_manager) {
                $scope.idx[status.instance].isManager = true
            } else if (status.is_worker) {
                $scope.idx[status.instance].isManager = false
            } else {
                $scope.idx[status.instance].isManager = null
            }
            $scope.$apply();
        });

        socket.on('instance k8s status', function(status) {
            if (!$scope.idx[status.instance]) {
                return
            }
            if (status.is_manager) {
                $scope.idx[status.instance].isK8sManager = true
            } else if (status.is_worker) {
                $scope.idx[status.instance].isK8sManager = false
            } else {
                $scope.idx[status.instance].isK8sManager = null
            }
            $scope.$apply();
        });

        socket.on('instance docker ports', function(status) {
          if (!$scope.idx[status.instance]) {
              return
          }
          $scope.idx[status.instance].ports = status.ports;
          $scope.$apply();
        });

        socket.on('instance docker swarm ports', function(status) {
            for(var i in status.instances) {
                var instance = status.instances[i];
                if ($scope.idxByHostname[instance]) {
                    $scope.idxByHostname[instance].swarmPorts = status.ports;
                }
            }
            $scope.$apply();
        });

        $scope.socket = socket;


        // If instance is passed in URL, select it
        let inst = $scope.idx[$location.hash()];
        if (inst) {
            $scope.showInstance(inst);
        } else if($scope.instances.length > 0) {
            // if no instance has been passed, select the first.
            $scope.showInstance($scope.instances[0]);
        }
      }, function(response) {
        if (response.status == 404) {
          document.write('session not found');
          return
        }
      });
    }

    $scope.getProxyUrl = function(instance, port) {
      var url = 'http://' + instance.proxy_host + '-' + port + '.direct.' + $scope.host;

      return url;
    }

    $scope.showInstance = function(instance) {
      $scope.selectedInstance = instance;
      $location.hash(instance.name);
      if (!instance.term) {
          $timeout(function() {
              createTerminal(instance);
              TerminalService.setFontSize(TerminalService.getFontSize());
              instance.term.focus();
              $timeout(function() {
              }, 0, false);
          }, 0, false);
          return
      }
      instance.term.focus();
    }

    $scope.removeInstance = function(name) {
        if ($scope.idx[name]) {
            var handler = $scope.idx[name].terminalBufferInterval;
            clearInterval(handler);
        }
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
    };

    $scope.openEditor = function(instance) {
      var w = window.screen.availWidth * 45  / 100;
      var h = window.screen.availHeight * 45  / 100;
      $window.open('/sessions/' + instance.session_id + '/instances/'+instance.name+'/editor', 'editor',
        'width='+w+',height='+h+',resizable,scrollbars=yes,status=1');
    };

    $scope.loadPlaygroundConf();
    $scope.getSession($scope.sessionId);

    $scope.createBuilderTerminal = function() {
      var builderTerminalContainer = document.getElementById('builder-terminal');
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

        /*
      setTimeout(function() {
        $scope.resize(term.proposeGeometry());
      }, 4);
      */

      instance.terminalBuffer = '';
      instance.terminalBufferInterval = setInterval(function() {
          if (instance.terminalBuffer.length > 0) {
              $scope.socket.emit('instance terminal in', instance.name, instance.terminalBuffer);
              instance.terminalBuffer = '';
          }
      }, 70);
      term.on('data', function(d) {
          instance.terminalBuffer += d;
      });

      instance.term = term;

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
  .config(['$mdIconProvider', '$locationProvider', '$mdThemingProvider', function($mdIconProvider, $locationProvider, $mdThemingProvider) {
    $locationProvider.html5Mode({enabled: true, requireBase: false});
    $mdIconProvider.defaultIconSet('../assets/social-icons.svg', 24);
    $mdThemingProvider.theme('kube')
      .primaryPalette('grey')
      .accentPalette('grey');
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
