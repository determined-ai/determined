:orphan:

** New Features**

-  Master Configuration: Add `always_redirect` option in OIDC and SAML configurations. When enabled, this option
      bypasses the standard Determined login screen and directly routes users to the configured SSO
      provider. This redirection persists unless the user explicitly signs out within the WebUI.

**Improvements**

- WebUI: Redirect SSO users to the SSO provider's authentication URIs when their session token has expired,
- ```
      instead of displaying the Determined sign-in page.
