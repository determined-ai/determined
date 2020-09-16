import { CopyOutlined } from '@ant-design/icons';
import { Button, notification, Result } from 'antd';
import React, { useCallback } from 'react';

import { globalStorage } from 'globalStorage';
import { copyToClipboard } from 'utils/dom';

import css from './AuthToken.module.scss';

const AuthToken: React.FC = () => {
  const token = globalStorage.getAuthToken || 'Auth token not found.';

  const handleCopyToClipboard = useCallback(async () => {
    try {
      await copyToClipboard(token);
      notification.open({
        description: 'Auth token copied to the clipboard.',
        message: 'Auth Token Copied',
      });
    } catch (e) {
      notification.warn({
        description: e.message,
        message: 'Unable to Copy to Clipboard',
      });
    }
  }, [ token ]);

  return (
    <Result
      className={css.base}
      extra={[
        <Button
          icon={<CopyOutlined />}
          key="copy"
          type="primary"
          onClick={handleCopyToClipboard}>
          Copy token to clipboard
        </Button>,
      ]}
      status="success"
      subTitle={token}
      title="Your Determined Authentication Token"
    />
  );
};

export default AuthToken;
