import { CopyOutlined } from '@ant-design/icons';
import { Button, Result } from 'antd';
import React, { useCallback } from 'react';

import { getCookie } from 'utils/browser';

import css from './AuthToken.module.scss';

const AuthToken: React.FC = () => {
  const token = getCookie('auth') || 'Auth token not found.';

  const handleCopyToClipboard = useCallback(
    (): Promise<void> => navigator.clipboard.writeText(token),
    [ token ],
  );

  return (
    <Result
      className={css.base}
      extra={[
        <Button href="/det/dashboard" key="dashboard" type="primary">
          Go to dashboard
        </Button>,
        <Button icon={<CopyOutlined />}
          key="copy" type="primary"
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
