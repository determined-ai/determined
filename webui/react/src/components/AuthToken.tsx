import { CopyOutlined } from '@ant-design/icons';
import { Result } from 'antd';
import React, { useCallback } from 'react';

import css from 'components/AuthToken.module.scss';
import Button from 'components/kit/Button';
import { globalStorage } from 'globalStorage';
import { notification } from 'utils/dialogApi';
import { copyToClipboard } from 'utils/dom';

const AuthToken: React.FC = () => {
  const token = globalStorage.authToken || 'Auth token not found.';

  const handleCopyToClipboard = useCallback(async () => {
    try {
      await copyToClipboard(token);
      notification.open({
        description: 'Auth token copied to the clipboard.',
        message: 'Auth Token Copied',
      });
    } catch (e) {
      notification.warning({
        description: (e as Error)?.message,
        message: 'Unable to Copy to Clipboard',
      });
    }
  }, [token]);

  return (
    <Result
      className={css.base}
      extra={[
        <Button icon={<CopyOutlined />} key="copy" type="primary" onClick={handleCopyToClipboard}>
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
