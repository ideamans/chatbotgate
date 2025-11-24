package middleware

// logoutTemplate is the HTML template for the logout page
const logoutTemplate = `<!DOCTYPE html>
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
			<div class="alert alert-success" style="text-align: left; margin-bottom: var(--spacing-md);">{{.Message}}</div>
			<a href="{{.LoginURL}}" class="btn btn-primary" style="width: 100%; margin-top: var(--spacing-md);">{{.LoginLabel}}</a>
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="{{.CreditIcon}}" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
	</div>
</div>
</body>
</html>`
