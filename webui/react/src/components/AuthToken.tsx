import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import { useToast } from 'hew/Toast';
import React, { useCallback } from 'react';

import { globalStorage } from 'globalStorage';
import { copyToClipboard } from 'utils/dom';

const AuthToken: React.FC = () => {
  const { openToast } = useToast();
  const token = globalStorage.authToken || 'Auth token not found.';

  const handleCopyToClipboard = useCallback(async () => {
    try {
      await copyToClipboard(token);
      openToast({
        description: 'Auth token copied to the clipboard.',
        title: 'Auth Token Copied',
      });
    } catch (e) {
      openToast({
        description: (e as Error)?.message,
        severity: 'Warning',
        title: 'Unable to Copy to Clipboard',
      });
    }
  }, [token, openToast]);

  return (
    <Message
      action={
        <Button
          icon={<Icon decorative name="clipboard" />}
          key="copy"
          type="primary"
          onClick={handleCopyToClipboard}>
          Copy token to clipboard
        </Button>
      }
      description={token}
      icon="checkmark"
      title="Your Determined Authentication Token"
    />
  );
};

export default AuthToken;
