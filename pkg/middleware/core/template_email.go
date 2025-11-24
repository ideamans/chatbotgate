package middleware

// emailSentTemplate is the HTML template for the email sent confirmation page
const emailSentTemplate = `<!DOCTYPE html>
<html lang="{{.Lang}}"{{if eq .Theme "dark"}} class="dark"{{else if eq .Theme "light"}} class="light"{{end}}>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}} - {{.ServiceName}}</title>
{{.StyleLinks}}
</head>
<body>
<div class="auth-container">
	<div style="width: 100%; max-width: 28rem;">
		<div class="card auth-card">
			{{.Header}}
			{{if .Subtitle}}
			<h2 class="auth-subtitle">{{.Subtitle}}</h2>
			{{end}}
			<div class="alert alert-success" style="text-align: left; margin-bottom: var(--spacing-md);">{{.Message}} {{.Detail}}</div>

			<!-- OTP Input Section -->
			<div style="text-align: center; margin-top: var(--spacing-lg); margin-bottom: var(--spacing-lg);">
				<div style="margin-bottom: var(--spacing-sm);">
					<span style="color: var(--color-text-secondary); font-size: 0.875rem;">{{.OTPLabel}}</span>
				</div>
				<form method="POST" action="{{.VerifyOTPPath}}" style="display: flex; flex-direction: column; align-items: center; gap: var(--spacing-sm);">
					<input
						type="text"
						name="otp"
						id="otp-input"
						class="input"
						placeholder="{{.OTPPlaceholder}}"
						maxlength="14"
						autocomplete="off"
						style="text-align: center; font-family: 'Courier New', monospace; font-size: 1.125rem; font-weight: 600; letter-spacing: 0.05em; background-color: var(--color-bg-muted); border: 2px solid var(--color-border-default); max-width: 16rem; transition: border-color 0.2s ease, background-color 0.2s ease;">
					<button type="submit" id="verify-button" class="btn btn-primary" disabled style="max-width: 16rem; width: 100%;">
						{{.VerifyButton}}
					</button>
				</form>
			</div>

			<a href="{{.LoginURL}}" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">{{.BackLabel}}</a>
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="{{.CreditIcon}}" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
	</div>
</div>
<script>
(function() {
	const otpInput = document.getElementById('otp-input');
	const verifyButton = document.getElementById('verify-button');
	if (!otpInput || !verifyButton) return;

	function validateOTP(value) {
		const cleaned = value.replace(/[^A-Z0-9]/gi, '').toUpperCase();
		return cleaned.length === 12 && /^[A-Z0-9]{12}$/.test(cleaned);
	}

	function updateUI(isValid) {
		if (isValid) {
			// Input: Green border and background
			otpInput.style.borderColor = 'var(--color-success)';
			otpInput.style.backgroundColor = 'color-mix(in srgb, var(--color-success) 10%, var(--color-bg-muted))';

			// Button: Enable (keep btn-primary style)
			verifyButton.disabled = false;
		} else {
			// Input: Default style
			otpInput.style.borderColor = 'var(--color-border-default)';
			otpInput.style.backgroundColor = 'var(--color-bg-muted)';

			// Button: Disable
			verifyButton.disabled = true;
		}
	}

	otpInput.addEventListener('input', function() {
		updateUI(validateOTP(this.value));
	});
})();
</script>
</body>
</html>`
