package middleware

// loginTemplate is the HTML template for the login page
const loginTemplate = `<!DOCTYPE html>
<html lang="{{.Lang}}"{{if eq .Theme "dark"}} class="dark"{{else if eq .Theme "light"}} class="light"{{end}}>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}} - {{.ServiceName}}</title>
{{.StyleLinks}}
<style>
.settings-toggle {
	position: fixed;
	top: var(--spacing-md);
	right: var(--spacing-md);
	display: flex;
	flex-direction: row;
	gap: var(--spacing-md);
	align-items: center;
	z-index: 100;
	background-color: var(--color-bg-elevated);
	padding: var(--spacing-xs) var(--spacing-md);
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border-default);
}
.settings-toggle select {
	padding: var(--spacing-xs) var(--spacing-sm);
	border: none;
	background: transparent;
	color: var(--color-text-secondary);
	font-size: 0.875rem;
	cursor: pointer;
	appearance: none;
	-webkit-appearance: none;
	-moz-appearance: none;
}
.settings-toggle select:hover {
	color: var(--color-text-primary);
}
.settings-toggle select:focus {
	outline: none;
	color: var(--color-text-primary);
}
</style>
</head>
<body>
<div class="settings-toggle">
	<select id="theme-select" onchange="changeTheme(this.value)">
		<option value="auto"{{if eq .Theme "auto"}} selected{{end}}>{{.Translations.ThemeAuto}}</option>
		<option value="light"{{if eq .Theme "light"}} selected{{end}}>{{.Translations.ThemeLight}}</option>
		<option value="dark"{{if eq .Theme "dark"}} selected{{end}}>{{.Translations.ThemeDark}}</option>
	</select>
	<select id="lang-select" onchange="changeLanguage(this.value)">
		<option value="en"{{if eq .Lang "en"}} selected{{end}}>{{.Translations.LanguageEn}}</option>
		<option value="ja"{{if eq .Lang "ja"}} selected{{end}}>{{.Translations.LanguageJa}}</option>
	</select>
</div>

<div class="auth-container">
	<div style="width: 100%; max-width: 28rem;">
		<div class="card auth-card">
			{{.Header}}
			<p class="auth-description">{{.ServiceDescription}}</p>
			{{if .Providers}}
			<div style="margin-bottom: var(--spacing-lg);">
				{{range .Providers}}
				<a href="{{.URL}}" class="btn btn-secondary provider-btn">
					<img src="{{.IconPath}}" alt="{{.Name}}">
					{{.Label}}
				</a>
				{{end}}
			</div>
			{{end}}
			{{if .EmailEnabled}}
			{{if .Providers}}
			<div class="auth-divider"><span>{{.Translations.Or}}</span></div>
			{{end}}
			<form method="POST" action="{{.EmailSendPath}}" id="email-form">
				<div class="form-group">
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--spacing-xs);">
						<label class="label" for="email" style="margin-bottom: 0;">{{.Translations.EmailLabel}}</label>
						<label style="display: flex; align-items: center; gap: 0.25rem; cursor: pointer; font-size: 0.875rem; color: var(--color-text-secondary);">
							<input type="checkbox" id="save-email-checkbox" style="cursor: pointer;">
							<span>{{.Translations.EmailSave}}</span>
						</label>
					</div>
					<input type="email" id="email" name="email" class="input" placeholder="you@example.com" required>
				</div>
				<button type="submit" class="btn btn-primary provider-btn">
					<img src="{{.EmailIconPath}}" alt="Email">
					{{.Translations.EmailSubmit}}
				</button>
			</form>
			<script>
			(function() {
				const emailInput = document.getElementById('email');
				const saveCheckbox = document.getElementById('save-email-checkbox');
				const STORAGE_KEY_EMAIL = 'saved_email';
				const STORAGE_KEY_SAVE = 'save_email_enabled';

				// Load saved settings
				const savedEmail = localStorage.getItem(STORAGE_KEY_EMAIL);
				const saveEnabled = localStorage.getItem(STORAGE_KEY_SAVE) === 'true';

				if (savedEmail && saveEnabled) {
					emailInput.value = savedEmail;
					saveCheckbox.checked = true;
				} else if (saveEnabled) {
					saveCheckbox.checked = true;
				}

				// Save email on input change (if checkbox is checked)
				emailInput.addEventListener('input', function() {
					if (saveCheckbox.checked) {
						localStorage.setItem(STORAGE_KEY_EMAIL, emailInput.value);
					}
				});

				// Handle checkbox changes
				saveCheckbox.addEventListener('change', function() {
					if (saveCheckbox.checked) {
						// Save current email value and remember the checkbox state
						localStorage.setItem(STORAGE_KEY_EMAIL, emailInput.value);
						localStorage.setItem(STORAGE_KEY_SAVE, 'true');
					} else {
						// Clear saved email and checkbox state
						localStorage.removeItem(STORAGE_KEY_EMAIL);
						localStorage.removeItem(STORAGE_KEY_SAVE);
					}
				});
			})();
			</script>
			{{end}}
			{{if .PasswordEnabled}}
			{{if or .Providers .EmailEnabled}}
			<div class="auth-divider"><span>{{.Translations.Or}}</span></div>
			{{end}}
			{{.PasswordFormHTML}}
			{{end}}
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="{{.CreditIcon}}" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
	</div>
</div>
<script>
function setCookie(name, value, days) {
	var expires = "";
	if (days) {
		var date = new Date();
		date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
		expires = "; expires=" + date.toUTCString();
	}
	document.cookie = name + "=" + value + expires + "; path=/; SameSite=Lax";
}

function changeTheme(theme) {
	setCookie("theme", theme, 365);

	// Apply theme immediately without reload
	var html = document.documentElement;
	if (theme === "dark") {
		html.classList.add("dark");
	} else if (theme === "light") {
		html.classList.remove("dark");
	} else {
		// Auto - check system preference
		if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
			html.classList.add("dark");
		} else {
			html.classList.remove("dark");
		}
	}
}

function changeLanguage(lang) {
	setCookie("lang", lang, 365);
	window.location.reload();
}

// Listen for system theme changes
if (window.matchMedia) {
	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
		var savedTheme = getCookie("theme");
		if (!savedTheme || savedTheme === "auto") {
			document.documentElement.classList.toggle("dark", e.matches);
		}
	});
}

function getCookie(name) {
	var nameEQ = name + "=";
	var ca = document.cookie.split(';');
	for(var i=0; i < ca.length; i++) {
		var c = ca[i];
		while (c.charAt(0) == ' ') c = c.substring(1, c.length);
		if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length, c.length);
	}
	return null;
}
</script>
</body>
</html>`
