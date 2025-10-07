console.log('[Oktaws] Content script loaded on:', window.location.href);

const DEFAULT_PORT = 8765;
let cliPort = DEFAULT_PORT;

chrome.storage.local.get(['cliPort'], (result) => {
  if (result.cliPort) {
    cliPort = result.cliPort;
    console.log('[Oktaws] Using saved port:', cliPort);
  }
});

chrome.runtime.onMessage.addListener((message) => {
  if (message.type === 'UPDATE_PORT') {
    cliPort = message.port;
    console.log('[Oktaws] Port updated to:', cliPort);
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
    console.log('[Oktaws] CLI not running, skipping SAML interception');
    return false;
  }
}

async function sendSAMLToCLI(samlResponse) {
  const isRunning = await checkCLIRunning();
  if (!isRunning) {
    console.log('[Oktaws] CLI not running, not sending SAML');
    return;
  }

  const url = `http://localhost:${cliPort}/callback`;
  
  console.log('[Oktaws] Sending SAML to CLI:', url, 'Length:', samlResponse.length);
  
  fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded'
    },
    mode: 'no-cors',
    body: 'SAMLResponse=' + encodeURIComponent(samlResponse)
  })
  .then(response => {
    console.log('[Oktaws] ✓ SAML sent to CLI');
    chrome.runtime.sendMessage({ type: 'SAML_SENT', success: true });
  })
  .catch(error => {
    console.error('[Oktaws] Error sending SAML:', error);
  });
}

function interceptFormSubmission() {
  const samlForm = document.querySelector('form[name="saml-form"], form[action*="saml"]');
  
  if (samlForm) {
    console.log('[Oktaws] Found SAML form:', samlForm);
    
    const samlInput = samlForm.querySelector('input[name="SAMLResponse"]');
    if (samlInput && samlInput.value) {
      console.log('[Oktaws] ✓ SAML found in form! Length:', samlInput.value.length);
      sendSAMLToCLI(samlInput.value);
      return true;
    }
  }
  
  const allForms = document.querySelectorAll('form');
  for (const form of allForms) {
    const samlInput = form.querySelector('input[name="SAMLResponse"]');
    if (samlInput && samlInput.value) {
      console.log('[Oktaws] ✓ SAML found in form! Length:', samlInput.value.length);
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
    console.log('[Oktaws] ✓ Intercepting form submit with SAML!');
    sendSAMLToCLI(samlInput.value);
  }
}, true);
