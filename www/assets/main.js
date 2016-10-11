//var term,
    //protocol,
    //socketURL,
    //socket,
    //pid,
    //charWidth,
    //charHeight;

//var terminalContainer = document.getElementById('terminal-container'),
    //optionElements = {
      //cursorBlink: document.querySelector('#option-cursor-blink')
    //},
    //colsElement = document.getElementById('cols'),
    //rowsElement = document.getElementById('rows');

//function setTerminalSize () {
  //var cols = parseInt(colsElement.value),
      //rows = parseInt(rowsElement.value),
      //width = (cols * charWidth).toString() + 'px',
      //height = (rows * charHeight).toString() + 'px';

  //terminalContainer.style.width = width;
  //terminalContainer.style.height = height;
  //term.resize(cols, rows);
//}

//colsElement.addEventListener('change', setTerminalSize);
//rowsElement.addEventListener('change', setTerminalSize);

//optionElements.cursorBlink.addEventListener('change', createTerminal);

//createTerminal();
//
//
function createTerminal(name) {
  var terminalContainer = document.getElementById('terminal-' + name);
  // Clean terminal
  while (terminalContainer.children.length) {
    terminalContainer.removeChild(terminalContainer.children[0]);
  }
  var term = new Terminal({
    cursorBlink: false
  });
  var sessionId = location.pathname.substr(location.pathname.lastIndexOf("/")+1);
  protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';
  socketURL = protocol + location.hostname + ((location.port) ? (':' + location.port) : '') + '/sessions/' + sessionId + '/instances/' + name + '/attach';

  term.open(terminalContainer);

  socket = new WebSocket(socketURL);
  socket.onopen = runRealTerminal(term);

  return term;

}


function runRealTerminal(term) {
  term.attach(socket);
  term._initialized = true;
}
