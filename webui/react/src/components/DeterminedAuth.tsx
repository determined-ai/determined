import { ConfigProvider } from 'antd';
import React, { useCallback, useState } from 'react';

import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import Link from 'components/Link';
import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { login } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { isLoginFailure } from 'services/utils';
import useUI from 'shared/contexts/stores/UI';
import { ErrorType } from 'shared/utils/error';
import { StorageManager } from 'shared/utils/storage';
import authStore from 'stores/auth';
import permissionStore from 'stores/permissions';
import userStore from 'stores/users';
import handleError from 'utils/error';

import css from './DeterminedAuth.module.scss';

interface Props {
  canceler: AbortController;
}

interface FromValues {
  password?: string;
  username?: string;
}

const storage = new StorageManager({ basePath: '/DeterminedAuth', store: window.localStorage });
const STORAGE_KEY_LAST_USERNAME = 'lastUsername';

const buttonTheme = {
  components: {
    Button: {
      colorPrimary: 'var(--theme-brand)',
      colorPrimaryActive: 'var(--theme-brand-strong)',
      colorPrimaryHover: 'var(--theme-brand-weak)',
    },
  },
};

const DeterminedAuth: React.FC<Props> = ({ canceler }: Props) => {
  const { actions: uiActions } = useUI();
  const rbacEnabled = useFeature().isOn('rbac');
  const [isBadCredentials, setIsBadCredentials] = useState<boolean>(false);
  const [canSubmit, setCanSubmit] = useState<boolean>(!!storage.get(STORAGE_KEY_LAST_USERNAME));
  const [isSubmitted, setIsSubmitted] = useState<boolean>(false);

  const onFinish = useCallback(
    async (creds: FromValues): Promise<void> => {
      uiActions.showSpinner();
      setCanSubmit(false);
      setIsSubmitted(true);
      try {
        const { token, user } = await login(
          {
            password: creds.password || '',
            username: creds.username || '',
          },
          { signal: canceler.signal },
        );
        updateDetApi({ apiKey: `Bearer ${token}` });
        authStore.setAuth({ isAuthenticated: true, token });
        userStore.updateCurrentUser(user);
        if (rbacEnabled) {
          // Now that we have logged in user, fetch userAssignments and userRoles and place into store.
          permissionStore.fetch(canceler.signal);
        }
        storage.set(STORAGE_KEY_LAST_USERNAME, creds.username);
      } catch (e) {
        const isBadCredentialsSync = isLoginFailure(e);
        setIsBadCredentials(isBadCredentialsSync); // this is not a sync operation
        uiActions.hideSpinner();
        const actionMsg = isBadCredentialsSync ? 'check your username and password.' : 'retry.';
        if (isBadCredentialsSync) storage.remove(STORAGE_KEY_LAST_USERNAME);
        handleError(e, {
          isUserTriggered: true,
          publicMessage: `Failed to login. Please ${actionMsg}`,
          publicSubject: 'Login failed',
          silent: false,
          type: isBadCredentialsSync ? ErrorType.Input : ErrorType.Server,
        });
      } finally {
        setCanSubmit(true);
        setIsSubmitted(false);
      }
    },
    [canceler, uiActions, rbacEnabled],
  );

  const onValuesChange = useCallback((changes: FromValues, values: FromValues): void => {
    const hasUsername = !!values.username;
    setIsBadCredentials(false);
    setCanSubmit(hasUsername);
  }, []);

  const loginForm = (
    <Form
      initialValues={{ username: storage.getWithDefault(STORAGE_KEY_LAST_USERNAME, '') }}
      name="login"
      onFinish={onFinish}
      onValuesChange={onValuesChange}>
      <Form.Item
        name="username"
        rules={[
          {
            message: 'Please type in your username.',
            required: true,
          },
        ]}>
        <Input autoFocus placeholder="username" prefix={<Icon name="user-small" size="small" />} />
      </Form.Item>
      <Form.Item name="password">
        <Input.Password placeholder="password" prefix={<Icon name="lock" size="small" />} />
      </Form.Item>
      {isBadCredentials && (
        <p className={[css.errorMessage, css.message].join(' ')}>Incorrect username or password.</p>
      )}
      <Form.Item>
        <ConfigProvider theme={buttonTheme}>
          <Button disabled={!canSubmit} htmlType="submit" loading={isSubmitted} type="primary">
            Sign In
          </Button>
        </ConfigProvider>
      </Form.Item>
    </Form>
  );

  return (
    <div className={css.base}>
      {loginForm}
      <p className={css.message}>
        Forgot your password, or need to manage users? Check out our&nbsp;
        <Link external path={paths.docs('/sysadmin-basics/users.html')} popout>
          docs
        </Link>
      </p>
    </div>
  );
};

export default DeterminedAuth;
