console.log('[Oktaws Background] Service worker loaded');

const DEFAULT_PORT = 8765;

chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('[Oktaws Background] Extension installed');
    chrome.storage.local.set({ cliPort: DEFAULT_PORT });
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  console.log('[Oktaws Background] Message received:', message.type);
  
  if (message.type === 'SAML_SENT') {
    console.log('[Oktaws Background] SAML sent successfully');
  }
  
  sendResponse({received: true});
  return true;
});
