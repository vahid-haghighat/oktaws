const DEFAULT_PORT = 8765;
let cliPort = DEFAULT_PORT;

chrome.storage.local.get(['cliPort'], (result) => {
  if (result.cliPort) {
    cliPort = result.cliPort;
  }
});

chrome.runtime.onMessage.addListener((message) => {
  if (message.type === 'UPDATE_PORT') {
    cliPort = message.port;
  }
});

async function checkCLIRunning() {
  try {
    const response = await fetch(`http://localhost:${cliPort}/status`, {
      method: 'GET',
      mode: 'no-cors'
    });
    return true;
  } catch (error) {
    return false;
  }
}

async function sendSAMLToCLI(samlResponse) {
  const isRunning = await checkCLIRunning();
  if (!isRunning) {
    return;
  }

  const url = `http://localhost:${cliPort}/callback`;

  fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded'
    },
    mode: 'no-cors',
    body: 'SAMLResponse=' + encodeURIComponent(samlResponse)
  })
  .then(response => {
    chrome.runtime.sendMessage({ type: 'SAML_SENT', success: true });
  })
  .catch(error => {
      // Do nothing
  });
}

function interceptFormSubmission() {
  const samlForm = document.querySelector('form[name="saml-form"], form[action*="saml"]');
  
  if (samlForm) {
    const samlInput = samlForm.querySelector('input[name="SAMLResponse"]');
    if (samlInput && samlInput.value) {
      sendSAMLToCLI(samlInput.value);
      return true;
    }
  }
  
  const allForms = document.querySelectorAll('form');
  for (const form of allForms) {
    const samlInput = form.querySelector('input[name="SAMLResponse"]');
    if (samlInput && samlInput.value) {
      sendSAMLToCLI(samlInput.value);
      return true;
    }
  }
  
  return false;
}

function observeForSAML() {
  if (interceptFormSubmission()) {
    return;
  }
  
  const observer = new MutationObserver((mutations) => {
    if (interceptFormSubmission()) {
      observer.disconnect();
    }
  });
  
  observer.observe(document.documentElement, {
    childList: true,
    subtree: true
  });
  
  setTimeout(() => {
    observer.disconnect();
  }, 10000);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', observeForSAML);
} else {
  observeForSAML();
}

document.addEventListener('submit', (e) => {
  const form = e.target;
  const samlInput = form.querySelector('input[name="SAMLResponse"]');
  if (samlInput && samlInput.value) {
    sendSAMLToCLI(samlInput.value);
  }
}, true);
