(function (window) {

    // declare
    var pwd = function () {
        this.instances = {};
        return;
    };

    // your sdk init function
    pwd.prototype.init = function (sessionId, opts) {
      var self = this;
      opts = opts || {};
      this.sessionId = sessionId;
      this.baseUrl = opts.baseUrl || window.location.origin;
      this.socket = io(this.baseUrl, {path: '/sessions/' + sessionId + '/ws' });
      this.socket.on('terminal out', function(name ,data) {
        var instance = self.instances[name];
        if (instance && instance.terms) {
          instance.terms.forEach(function(term) {term.write(data)});
        }
      });
    };

    pwd.prototype.createInstance = function(callback) {
      var self = this;
      //TODO handle http connection errors
      var request = new XMLHttpRequest();
      request.open('POST', self.baseUrl + '/sessions/' + this.sessionId + '/instances' , true);
      request.onload = function() {
        if (request.status == 200) {
          var i = JSON.parse(request.responseText);
          i.terms = [];
          self.instances[i.name] = i;
          callback(undefined, i);
        } else if (request.status == 409) {
          var err = new Error();
          err.max = true;
          callback(err);
        } else {
          callback(new Error());
        }
      };
      request.send();
    }

    pwd.prototype.createTerminal = function(selector, name) {
        var self = this;
        var i = this.instances[name];
        if (!i) {
          i = {name: name, terms: []};
          this.instances[name] = i;
        }
        var elements = document.querySelectorAll(selector);
        elements.forEach(function(el) {
          var term = new Terminal({cursorBlink: false});
          term.open(el);
          term.on('data', function(d) {
            self.socket.emit('terminal in', i.name, d);
          });
          i.terms.push(term);
        });

        var actions = document.querySelectorAll('[for="'+selector+'"]');
        actions.forEach(function(actionEl) {
          actionEl.onclick = function() {
            self.socket.emit('terminal in', i.name, this.innerText);
          };
        });
        return i.terms;
    }

    pwd.prototype.terminal = function(selector, callback) {
      var self = this;
      this.createInstance(function(err, instance) {
          if (err && err.max) {
            !callback || callback(new Error("Max instances reached"))
            return
          } else if (err) {
            !callback || callback(new Error("Error creating instance"))
            return
          }

          self.createTerminal(selector, instance.name);

          !callback || callback(undefined, instance);

      });
    }



    // define your namespace myApp
    window.pwd = new pwd();

})(window, undefined);
