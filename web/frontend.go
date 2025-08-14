package web

// Embed the frontend HTML as a Go string
var FrontendHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Delete Computer - OCS Inventory</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    body { background: #18171c; color: #e6e6e6; }
    .ocs-modal-bg { background: rgba(0,0,0,0.7); }
    .ocs-box { border: 2px solid #e6e6e6; background: transparent; color: #e6e6e6; }
    .ocs-input, .ocs-btn, .ocs-checkbox { background: transparent; border: 2px solid #e6e6e6; color: #e6e6e6; }
    .ocs-input:focus { outline: none; border-color: #93318e; }
    .ocs-btn { border-radius: 8px; padding: 0.5rem 1.5rem; font-weight: bold; transition: background 0.2s, color 0.2s; }
    .ocs-btn { background: #93318e; color: #e6e6e6; border-color: #93318e; }
    .ocs-btn:hover { background: #e6e6e6; color: #93318e; }
    .ocs-checkbox:checked { accent-color: #93318e; }
    .ocs-title { color: #e6e6e6; font-size: 2rem; font-weight: bold; margin-bottom: 1rem; }
    .ocs-label { color: #e6e6e6; }
    .ocs-logo { border: 2px solid #e6e6e6; border-radius: 9999px; width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; margin-bottom: 1rem; }
    .ocs-logo-text { color: #93318e; font-size: 2rem; font-weight: bold; }
    .ocs-step { border: 2px solid #e6e6e6; border-radius: 1rem; padding: 2rem; background: rgba(24,23,28,0.95); min-width: 320px; max-width: 350px; }
    .ocs-success-check { color: #93318e; width: 48px; height: 48px; margin-bottom: 1rem; }
    .ocs-btn:disabled { opacity: 0.5; cursor: not-allowed; }
    .ocs-link { color: #93318e; text-decoration: underline; }
    .ocs-error { color: #ff6b6b; font-size: 0.95rem; margin-top: 0.5rem; }
  </style>
</head>
<body class="min-h-screen flex items-center justify-center">
  <div class="fixed inset-0 ocs-modal-bg flex items-center justify-center z-10">
    <!-- Step 1: Login -->
    <div id="stepLogin" class="ocs-step flex flex-col items-center">
      <div class="ocs-logo"><span class="ocs-logo-text">OCS</span></div>
      <div class="ocs-title">Sign-in to OCS</div>
      <form id="loginForm" class="w-full flex flex-col gap-3">
        <input id="username" class="ocs-input rounded px-3 py-2" type="text" placeholder="Username" required autofocus autocomplete="username">
        <input id="password" class="ocs-input rounded px-3 py-2" type="password" placeholder="Password" required autocomplete="current-password">
        <button type="submit" class="ocs-btn mt-2">Login</button>
        <div id="loginError" class="ocs-error hidden"></div>
      </form>
    </div>
    <!-- Step 2: Confirm -->
    <div id="stepConfirm" class="ocs-step flex flex-col items-center hidden">
      <div class="ocs-logo"><span class="ocs-logo-text">OCS</span></div>
      <div class="ocs-title">Delete Computer</div>
      <div class="text-center mb-2">
        You are about to delete computer <span class="font-bold" id="compName"></span> from OCS Inventory.<br>
        Please complete validation steps below.
      </div>
      <form id="confirmForm" class="w-full flex flex-col gap-3 mt-2">
        <div class="flex items-center gap-2 justify-center">
          <span id="captchaQ" class="font-semibold"></span>
          <span>=</span>
          <input id="captchaA" class="ocs-input rounded px-2 py-1 w-20 text-center" type="text" required autocomplete="off">
        </div>
        <label class="flex items-center gap-2">
          <input id="confirmCheck" type="checkbox" class="ocs-checkbox" required>
          <span class="text-sm">I Understand and confirm this deletion</span>
        </label>
        <button type="submit" class="ocs-btn mt-2">Delete</button>
        <div id="confirmError" class="ocs-error hidden"></div>
      </form>
    </div>
    <!-- Step 3: Success -->
    <div id="stepSuccess" class="ocs-step flex flex-col items-center hidden">
      <svg class="ocs-success-check" viewBox="0 0 24 24" fill="none" stroke="#93318e" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5 13l4 4L19 7"/></svg>
      <div class="ocs-title" style="color:#93318e;">Success</div>
      <div class="text-center mb-4" id="successMsg"></div>
      <button onclick="location.reload()" class="ocs-btn">OK</button>
    </div>
  </div>
  <script>
    // Get computer name from query param
    function getQueryParam(name) {
      const url = new URL(window.location.href);
      return url.searchParams.get(name);
    }
    const compName = getQueryParam('name') || '';
    document.getElementById('compName').textContent = compName;

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

    // Login form
    document.getElementById('loginForm').onsubmit = async function(e) {
      e.preventDefault();
      const username = document.getElementById('username').value.trim();
      const password = document.getElementById('password').value;
      const errDiv = document.getElementById('loginError');
      errDiv.classList.add('hidden');
      try {
        const res = await fetch('/auth-token', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Login failed');
        jwtToken = data.token;
        // Simpan token ke cookie agar bisa diakses di POST
        document.cookie = 'ocsjwt=' + jwtToken + '; path=/; max-age=180; SameSite=Strict';
        showStep('stepConfirm');
      } catch (err) {
        errDiv.textContent = err.message;
        errDiv.classList.remove('hidden');
      }
    };

    // Confirm form
    document.getElementById('confirmForm').onsubmit = async function(e) {
      e.preventDefault();
      const answer = document.getElementById('captchaA').value.trim();
      const errDiv = document.getElementById('confirmError');
      errDiv.classList.add('hidden');
      if (parseInt(answer) !== captchaX + captchaY) {
        errDiv.textContent = 'Captcha salah!';
        errDiv.classList.remove('hidden');
        return;
      }
      if (!document.getElementById('confirmCheck').checked) {
        errDiv.textContent = 'Anda harus konfirmasi penghapusan.';
        errDiv.classList.remove('hidden');
        return;
      }
      try {
        // Ambil token dari cookie
        let jwtToken = '';
        document.cookie.split(';').forEach(function(c) {
          let [k,v] = c.trim().split('=');
          if (k === 'ocsjwt') jwtToken = v;
        });
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
        errDiv.textContent = err.message;
        errDiv.classList.remove('hidden');
      }
    };
  </script>
</body>
</html>
`
