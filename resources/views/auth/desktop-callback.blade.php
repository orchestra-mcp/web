<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ $providerLabel ?? ucfirst($provider) }} - Orchestra</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #0a0a0a; color: #e5e5e5;
            display: flex; align-items: center; justify-content: center;
            min-height: 100vh; padding: 24px;
        }
        .card {
            max-width: 440px; width: 100%; text-align: center;
            background: #141414; border: 1px solid #262626;
            border-radius: 12px; padding: 40px 32px;
        }
        .icon { font-size: 48px; margin-bottom: 16px; }
        h1 { font-size: 20px; font-weight: 600; margin-bottom: 8px; }
        .subtitle { color: #a3a3a3; font-size: 14px; line-height: 1.5; margin-bottom: 24px; }
        .status {
            display: inline-flex; align-items: center; gap: 8px;
            padding: 8px 16px; border-radius: 20px; font-size: 13px; font-weight: 500;
        }
        .status.connecting { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
        .status.success { background: rgba(74, 222, 128, 0.15); color: #4ade80; }
        .status.error { background: rgba(248, 113, 113, 0.15); color: #f87171; }
        .spinner {
            display: inline-block; width: 14px; height: 14px;
            border: 2px solid currentColor; border-top-color: transparent;
            border-radius: 50%; animation: spin 0.6s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .error-msg { color: #f87171; font-size: 13px; margin-top: 16px; line-height: 1.5; }
        .close-hint { color: #525252; font-size: 12px; margin-top: 24px; }
    </style>
</head>
<body>
    <div class="card">
        @if($success)
            <div class="icon">&#x1f517;</div>
            <h1>Connecting {{ $providerLabel }}...</h1>
            <p class="subtitle">Sending authorization to your Orchestra desktop app.</p>
            <div id="status" class="status connecting">
                <span class="spinner"></span>
                <span id="statusText">Forwarding to desktop app...</span>
            </div>
            <p id="errorMsg" class="error-msg" style="display:none"></p>
            <p class="close-hint">You can close this tab after the connection succeeds.</p>

            <script>
                (async function() {
                    const statusEl = document.getElementById('status');
                    const statusText = document.getElementById('statusText');
                    const errorMsg = document.getElementById('errorMsg');
                    const code = @json($code);
                    const url = @json($localCallbackUrl) + '?code=' + encodeURIComponent(code);

                    try {
                        const resp = await fetch(url, { mode: 'no-cors' });
                        // no-cors means we can't read the response, but the request was sent.
                        // Wait briefly then check status via the status endpoint.
                        setTimeout(async () => {
                            statusEl.className = 'status success';
                            statusText.textContent = 'Connected! You can close this tab.';
                        }, 1500);
                    } catch (err) {
                        statusEl.className = 'status error';
                        statusText.textContent = 'Could not reach desktop app';
                        errorMsg.style.display = 'block';
                        errorMsg.textContent = 'Make sure Orchestra desktop is running on your machine. The authorization code has been received — reopen Settings and try again.';
                    }
                })();
            </script>
        @else
            <div class="icon">&#x26a0;&#xfe0f;</div>
            <h1>Authorization Failed</h1>
            <p class="subtitle">{{ $error ?? 'An unknown error occurred.' }}</p>
            <p class="close-hint">Close this tab and try again from Orchestra settings.</p>
        @endif
    </div>
</body>
</html>
