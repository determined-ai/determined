import Button from 'determined-ui/Button';
import Icon from 'determined-ui/Icon';
import Message from 'determined-ui/Message';
import { makeToast } from 'determined-ui/Toast';
import React, { useCallback, useRef } from 'react';

import { globalStorage } from 'globalStorage';
import { copyToClipboard } from 'utils/dom';

const AuthToken: React.FC = () => {
  const token = globalStorage.authToken || 'Auth token not found.';
  const containerRef = useRef(null);
  const handleCopyToClipboard = useCallback(async () => {
    try {
      await copyToClipboard(token);
      makeToast({
        containerRef,
        description: 'Auth token copied to the clipboard.',
        title: 'Auth Token Copied',
      });
    } catch (e) {
      makeToast({
        containerRef,
        description: (e as Error)?.message,
        severity: 'Warning',
        title: 'Unable to Copy to Clipboard',
      });
    }
  }, [token]);

  return (
    <div ref={containerRef}>
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
    </div>
  );
};

export default AuthToken;
