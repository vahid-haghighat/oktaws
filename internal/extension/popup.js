// Popup script for extension settings

// Load current port
chrome.storage.local.get(['cliPort'], (result) => {
  if (result.cliPort) {
    document.getElementById('port').value = result.cliPort;
  }
});

// Save port button
document.getElementById('savePort').addEventListener('click', () => {
  const port = parseInt(document.getElementById('port').value);
  
  if (port < 1024 || port > 65535) {
    alert('Port must be between 1024 and 65535');
    return;
  }
  
  chrome.storage.local.set({ cliPort: port }, () => {
    // Update all tabs
    chrome.tabs.query({}, (tabs) => {
      tabs.forEach(tab => {
        chrome.tabs.sendMessage(tab.id, {
          type: 'UPDATE_PORT',
          port: port
        }).catch(() => {});
      });
    });
    
    // Show success
    const btn = document.getElementById('savePort');
    btn.textContent = '✓ Saved!';
    btn.style.background = '#10b981';
    
    setTimeout(() => {
      btn.textContent = 'Save Port';
      btn.style.background = '#667eea';
    }, 2000);
  });
});

