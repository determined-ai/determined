:orphan:

**Breaking Changes**

-  Python SDK and CLI: Password requirements are now enforced for all non-remote users. For more
   information, visit :ref:`password-requirements`.

   -  This change affects the ``create_user``, ``User.change_password``, ``det user create``, and
      ``det user change-password`` commands.
   -  Existing users with empty or non-compliant passwords can still sign in. However, we recommend
      updating these passwords to meet the new requirements as soon as possible.

-  CLI: When creating non-remote users with ``det user create``, setting a password is now
   mandatory.

   -  You can set the password interactively by following the prompts during user creation.
   -  Alternatively, set the password non-interactively using the ``--password`` option.
   -  This requirement does not apply to users created with the ``--remote`` option, as Single
      Sign-On will be used for these users.
