/* This will replace ./swagger-ui-main.js for static, cloud deployed versions of docs */

window.onload = function() {
  // Begin Swagger UI call region
  const ui = SwaggerUIBundle({
    url: 'api.swagger.json',
    supportedSubmitMethods: [],
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout"
  })
  // End Swagger UI call region

  window.ui = ui
}
