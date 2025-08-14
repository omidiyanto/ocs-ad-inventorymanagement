package web

// Embed the frontend HTML as a Go string
var FrontendHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Delete Computer - OCS Inventory</title>
  <style>
    /* Light Mode Palette & Base */
    :root {
      --bg-page: #f9fafb; /* Off-white */
      --bg-card: #ffffff; /* Pure white */
      --text-primary: #1f2937; /* Dark Gray */
      --text-secondary: #6b7280; /* Medium Gray */
      --border-color: #d1d5db; /* Light Gray */
      --accent-purple: #93318e;
      --accent-purple-hover: #7a2876; /* Darker purple */
      --error-color: #be123c; /* Rose Red */
    }
    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }
    html {
      font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';
    }
    body {
      background: var(--bg-page);
      color: var(--text-primary);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 1rem;
    }
    .hidden {
      display: none !important;
    }

    /* Main Container & Steps */
    .ocs-modal-bg {
      position: fixed;
      inset: 0;
      background: rgba(0,0,0,0.5);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 10;
    }
    .ocs-step {
      border: 1px solid var(--border-color);
      border-radius: 1rem;
      padding: 2rem;
      background: var(--bg-card);
      width: 95%;
      max-width: 380px; /* Slightly larger for better spacing */
      display: flex;
      flex-direction: column;
      align-items: center;
      box-shadow: 0 4px 12px rgba(0,0,0,0.08);
    }

    /* Logo and Titles */
    .ocs-logo {
      border: 2px solid var(--accent-purple);
      border-radius: 9999px;
      width: 64px;
      height: 64px;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 1rem;
    }
    .ocs-logo-text {
      color: var(--accent-purple);
      font-size: 2rem;
      font-weight: bold;
    }
    .ocs-title {
      color: var(--text-primary);
      font-size: 1.875rem; /* Slightly adjusted size */
      font-weight: bold;
      margin-bottom: 1rem;
      text-align: center;
    }

    /* Forms and Inputs */
    .ocs-form {
      width: 100%;
      display: flex;
      flex-direction: column;
      gap: 0.85rem;
    }
    .ocs-input {
      background: var(--bg-card);
      border: 1px solid var(--border-color);
      color: var(--text-primary);
      border-radius: 8px; /* Softer radius */
      padding: 0.65rem 0.75rem;
      width: 100%;
      transition: border-color 0.2s, box-shadow 0.2s;
    }
    .ocs-input:focus {
      outline: none;
      border-color: var(--accent-purple);
      box-shadow: 0 0 0 3px rgba(147, 49, 142, 0.2);
    }
    .ocs-label {
      color: var(--text-secondary);
      display: flex;
      align-items: center;
      gap: 0.5rem;
      cursor: pointer;
    }
    .ocs-checkbox {
      width: 1em;
      height: 1em;
      accent-color: var(--accent-purple);
    }
    .ocs-btn {
      background: var(--accent-purple);
      color: #ffffff;
      border: 1px solid var(--accent-purple);
      border-radius: 8px;
      padding: 0.75rem 1.5rem;
      font-weight: bold;
      transition: background-color 0.2s;
      cursor: pointer;
      margin-top: 0.5rem;
    }
    .ocs-btn:hover {
      background: var(--accent-purple-hover);
      border-color: var(--accent-purple-hover);
    }
    .ocs-btn:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    /* Specific Step Content */
    .ocs-delete-info {
      text-align: center;
      margin-bottom: 1rem;
      line-height: 1.5;
      color: var(--text-secondary);
    }
    .ocs-captcha-container {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      margin-top: 0.5rem;
    }
    #captchaA {
      width: 5rem;
      text-align: center;
    }
    .ocs-confirm-label {
      font-size: 0.875rem;
    }
    .ocs-success-check {
      color: var(--accent-purple);
      width: 48px;
      height: 48px;
      margin-bottom: 1rem;
    }
    #successMsg {
      text-align: center;
      margin-bottom: 1rem;
      font-size: 1.125rem;
      color: var(--text-primary);
    }
    .font-bold { font-weight: 600; }

    /* Custom Error Modal */
    .error-modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 100;
        padding: 1rem;
    }
    .error-modal-box {
        background: var(--bg-card);
        border: 1px solid var(--border-color);
        border-radius: 1rem;
        padding: 2rem;
        text-align: center;
        width: 95%;
        max-width: 400px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.1);
    }
    .error-modal-title {
        font-size: 1.5rem;
        font-weight: bold;
        margin-bottom: 1rem;
        color: var(--error-color);
    }
    .error-modal-text {
        margin-bottom: 1.5rem;
        line-height: 1.5;
        color: var(--text-secondary);
    }

    /* Responsive adjustments */
    @media (max-width: 400px) {
      .ocs-step, .error-modal-box {
        padding: 1.5rem;
      }
      .ocs-title {
        font-size: 1.5rem;
      }
    }
  </style>
</head>
<body>
  <div id="errorModal" class="error-modal-overlay hidden">
      <div class="error-modal-box">
          <div class="error-modal-title">Error</div>
          <p id="errorModalText" class="error-modal-text"></p>
          <button id="errorModalCloseBtn" class="ocs-btn">OK</button>
      </div>
  </div>

  <div class="ocs-modal-bg">
    <div id="stepLogin" class="ocs-step">
      <div class="ocs-logo"><span class="ocs-logo-text">OCS</span></div>
      <div class="ocs-title">Sign-in to OCS</div>
      <form id="loginForm" class="ocs-form">
        <input id="username" class="ocs-input" type="text" placeholder="Username" required autofocus autocomplete="username">
        <input id="password" class="ocs-input" type="password" placeholder="Password" required autocomplete="current-password">
        <button type="submit" class="ocs-btn">Login</button>
      </form>
    </div>

    <div id="stepConfirm" class="ocs-step hidden">
      <div class="ocs-logo"><span class="ocs-logo-text">OCS</span></div>
      <div class="ocs-title">Delete Computer</div>
      <div class="ocs-delete-info">
        You are about to delete computer <span class="font-bold" id="compName"></span> from OCS Inventory.<br>
        Please complete validation steps below.
      </div>
      <form id="confirmForm" class="ocs-form">
        <div class="ocs-captcha-container">
          <span id="captchaQ" class="font-bold"></span>
          <span>=</span>
          <input id="captchaA" class="ocs-input" type="text" required autocomplete="off">
        </div>
        <label class="ocs-label">
          <input id="confirmCheck" type="checkbox" class="ocs-checkbox" required>
          <span class="ocs-confirm-label">I Understand and confirm this deletion</span>
        </label>
        <button type="submit" class="ocs-btn">Delete</button>
      </form>
    </div>

    <div id="stepSuccess" class="ocs-step hidden">
      <svg class="ocs-success-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5 13l4 4L19 7"/></svg>
      <div class="ocs-title" style="color: var(--accent-purple);">Success</div>
      <div id="successMsg"></div>
      <button onclick="location.reload()" class="ocs-btn">OK</button>
    </div>
  </div>

  <script>
    // Custom Error Modal Logic
    const errorModal = document.getElementById('errorModal');
    const errorModalText = document.getElementById('errorModalText');
    const errorModalCloseBtn = document.getElementById('errorModalCloseBtn');

    function showError(msg) {
        errorModalText.innerHTML = msg;
        errorModal.classList.remove('hidden');
    }
    errorModalCloseBtn.onclick = function() {
        errorModal.classList.add('hidden');
    }
    window.onclick = function(event) {
        if (event.target == errorModal) {
            errorModal.classList.add('hidden');
        }
    }

    // Get computer name from query param
    function getQueryParam(name) {
      const url = new URL(window.location.href);
      return url.searchParams.get(name);
    }

    const compName = getQueryParam('name') || '';
    document.getElementById('compName').textContent = compName;
    if (!compName) {
      showError('Parameter ?name= wajib diisi di URL.');
      document.getElementById('stepLogin').style.display = 'none';
    }

    // Captcha
    let captchaX = Math.floor(Math.random()*10+1), captchaY = Math.floor(Math.random()*10+1);
    document.getElementById('captchaQ').textContent = captchaX + ' + ' + captchaY;

    // State
    let jwtToken = '';

    // Step control
    function showStep(step) {
      document.getElementById('stepLogin').classList.add('hidden');
      document.getElementById('stepConfirm').classList.add('hidden');
      document.getElementById('stepSuccess').classList.add('hidden');
      document.getElementById(step).classList.remove('hidden');
    }

    // Prevent skipping steps
    function enforceStep(step) {
      if (step === 'stepConfirm' && !jwtToken) {
        showError('Anda harus login terlebih dahulu.');
        showStep('stepLogin');
        return false;
      }
      if (step === 'stepSuccess' && !jwtToken) {
        showError('Akses tidak valid.');
        showStep('stepLogin');
        return false;
      }
      return true;
    }

    // Login form
    document.getElementById('loginForm').onsubmit = async function(e) {
      e.preventDefault();
      if (!compName) return showError('Parameter ?name= wajib diisi di URL.');
      const username = document.getElementById('username').value.trim();
      const password = document.getElementById('password').value;
      if (!username || !password) return showError('Username dan password wajib diisi.');

      try {
        const res = await fetch('/auth-token', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Login failed');
        jwtToken = data.token;
        document.cookie = 'ocsjwt=' + jwtToken + '; path=/; max-age=180; SameSite=Strict';
        showStep('stepConfirm');
      } catch (err) {
        showError(err.message);
      }
    };

    // Confirm form
    document.getElementById('confirmForm').onsubmit = async function(e) {
      e.preventDefault();
      if (!enforceStep('stepConfirm')) return;

      const answer = document.getElementById('captchaA').value.trim();
      if (parseInt(answer) !== captchaX + captchaY) {
        showError('Captcha salah!');
        return;
      }
      if (!document.getElementById('confirmCheck').checked) {
        showError('Anda harus konfirmasi penghapusan.');
        return;
      }

      try {
        let jwtToken = '';
        document.cookie.split(';').forEach(function(c) {
          let [k,v] = c.trim().split('=');
          if (k === 'ocsjwt') jwtToken = v;
        });

        if (!jwtToken) {
            showError('Session login tidak valid. Silakan login ulang.');
            showStep('stepLogin');
            return;
        }

        const res = await fetch('/delete-computer', {
          method: 'POST',
          headers: {
            'Authorization': 'Bearer ' + jwtToken,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ name: compName })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Delete failed');

        document.getElementById('successMsg').textContent = '"' + compName + '" Successfully Removed from OCS Inventory.';
        showStep('stepSuccess');

      } catch (err) {
        showError(err.message);
      }
    };

    // Prevent direct access to confirm/success without login
    window.addEventListener('DOMContentLoaded', function() {
      if (!compName) {
        showStep('stepLogin');
        return;
      }
      let hasToken = false;
      document.cookie.split(';').forEach(function(c) {
        let [k,v] = c.trim().split('=');
        if (k === 'ocsjwt' && v) {
            hasToken = true;
            jwtToken = v;
        }
      });

      if (hasToken) {
        showStep('stepConfirm');
      } else {
        showStep('stepLogin');
      }
    });
  </script>
</body>
</html>
`
