const DEFAULT_PORT = 8765;

chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    chrome.storage.local.set({ cliPort: DEFAULT_PORT });
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'SAML_SENT') {
    // Do nothing
  }
  
  sendResponse({received: true});
  return true;
});
