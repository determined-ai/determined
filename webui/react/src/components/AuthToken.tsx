import React, { useCallback } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import Message from 'components/kit/Message';
import { makeToast } from 'components/kit/Toast';
import { globalStorage } from 'globalStorage';
import { copyToClipboard } from 'utils/dom';

const AuthToken: React.FC = () => {
  const token = globalStorage.authToken || 'Auth token not found.';

  const handleCopyToClipboard = useCallback(async () => {
    try {
      await copyToClipboard(token);
      makeToast({
        description: 'Auth token copied to the clipboard.',
        title: 'Auth Token Copied',
      });
    } catch (e) {
      makeToast({
        description: (e as Error)?.message,
        severity: 'Warning',
        title: 'Unable to Copy to Clipboard',
      });
    }
  }, [token]);

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
