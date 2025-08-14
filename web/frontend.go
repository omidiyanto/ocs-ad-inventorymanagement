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
    /* General Styling */
    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }
    html {
      font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';
    }
    body {
      background: #18171c;
      color: #e6e6e6;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .hidden {
      display: none !important;
    }

    /* Main Container & Steps */
    .ocs-modal-bg {
      position: fixed;
      inset: 0;
      background: rgba(0,0,0,0.7);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 10;
    }
    .ocs-step {
      border: 2px solid #e6e6e6;
      border-radius: 1rem;
      padding: 2rem;
      background: rgba(24,23,28,0.95);
      min-width: 320px;
      max-width: 350px;
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    /* Logo and Titles */
    .ocs-logo {
      border: 2px solid #e6e6e6;
      border-radius: 9999px;
      width: 64px;
      height: 64px;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 1rem;
    }
    .ocs-logo-text {
      color: #93318e;
      font-size: 2rem;
      font-weight: bold;
    }
    .ocs-title {
      color: #e6e6e6;
      font-size: 2rem;
      font-weight: bold;
      margin-bottom: 1rem;
      text-align: center;
    }

    /* Forms and Inputs */
    .ocs-form {
      width: 100%;
      display: flex;
      flex-direction: column;
      gap: 0.75rem; /* 12px */
    }
    .ocs-input {
      background: transparent;
      border: 2px solid #e6e6e6;
      color: #e6e6e6;
      border-radius: 4px;
      padding: 0.5rem 0.75rem;
      width: 100%;
    }
    .ocs-input:focus {
      outline: none;
      border-color: #93318e;
    }
    .ocs-label {
      color: #e6e6e6;
      display: flex;
      align-items: center;
      gap: 0.5rem; /* 8px */
    }
    .ocs-checkbox {
      background: transparent;
      border: 2px solid #e6e6e6;
      color: #e6e6e6;
      accent-color: #93318e;
    }
    .ocs-btn {
      background: #93318e;
      color: #e6e6e6;
      border: 2px solid #93318e;
      border-radius: 8px;
      padding: 0.5rem 1.5rem;
      font-weight: bold;
      transition: background 0.2s, color 0.2s;
      cursor: pointer;
      margin-top: 0.5rem;
    }
    .ocs-btn:hover {
      background: #e6e6e6;
      color: #93318e;
    }
    .ocs-btn:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    /* Specific Step Content */
    .ocs-delete-info {
      text-align: center;
      margin-bottom: 0.5rem;
      line-height: 1.5;
    }
    .ocs-captcha-container {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      margin-top: 0.5rem;
    }
    #captchaA {
      width: 5rem; /* 80px */
      text-align: center;
      padding: 0.25rem 0.5rem;
    }
    .ocs-confirm-label {
      font-size: 0.875rem; /* 14px */
    }
    .ocs-success-check {
      color: #93318e;
      width: 48px;
      height: 48px;
      margin-bottom: 1rem;
    }
    #successMsg {
      text-align: center;
      margin-bottom: 1rem;
    }
    .font-bold { font-weight: bold; }

    /* Custom Error Modal */
    .error-modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 100;
    }
    .error-modal-box {
        background: #18171c;
        border: 2px solid #ff6b6b;
        border-radius: 1rem;
        padding: 2rem;
        color: #e6e6e6;
        text-align: center;
        min-width: 320px;
        max-width: 400px;
    }
    .error-modal-title {
        font-size: 1.5rem;
        font-weight: bold;
        margin-bottom: 1rem;
        color: #ff6b6b;
    }
    .error-modal-text {
        margin-bottom: 1.5rem;
        line-height: 1.5;
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
      <div class="ocs-title" style="color:#93318e;">Success</div>
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
    // Close modal if user clicks outside of it
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
        // Ambil token dari cookie
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
        // Error already shown, just ensure login step is visible
        showStep('stepLogin');
        return;
      }
      // If already have jwt in cookie, allow direct confirm
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
