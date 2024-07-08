:orphan:

** New Features**

-  Master Configuration: Add `always_redirect` option in OIDC and SAML configurations. When enabled, this option
      bypasses the standard Determined login screen and directly routes users to the configured SSO
      provider. This redirection persists unless the user explicitly signs out within the WebUI.

**Improvements**

-  WebUI: Redirect SSO users to SSO provider authentication URIs when the provided session token is expired,
      rather than showing the determined Sign-In page.
