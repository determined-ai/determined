import Button from 'hew/Button';
import Form from 'hew/Form';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import useUI from 'components/ThemeProvider';
import { paths } from 'routes/utils';
import { login } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { isLoginFailure } from 'services/utils';
import authStore from 'stores/auth';
import determinedStore from 'stores/determinedInfo';
import permissionStore from 'stores/permissions';
import userStore from 'stores/users';
import handleError, { ErrorLevel, ErrorType, handleWarning } from 'utils/error';
import { useObservable } from 'utils/observable';
import { StorageManager } from 'utils/storage';
import { isPasswordWeak } from 'utils/user';

import css from './DeterminedAuth.module.scss';

interface Props {
  canceler: AbortController;
}

interface FormValues {
  password?: string;
  username?: string;
}

export const WEAK_PASSWORD_SUBJECT = 'Weak Password';
export const USERNAME_ID = 'username';
export const PASSWORD_ID = 'password';
export const SUBMIT_ID = 'submit';

const storage = new StorageManager({ basePath: '/DeterminedAuth', store: window.localStorage });
const STORAGE_KEY_LAST_USERNAME = 'lastUsername';

const DeterminedAuth: React.FC<Props> = ({ canceler }: Props) => {
  const { actions: uiActions } = useUI();
  const { rbacEnabled } = useObservable(determinedStore.info);
  const [isBadCredentials, setIsBadCredentials] = useState<boolean>(false);
  const [canSubmit, setCanSubmit] = useState<boolean>(!!storage.get(STORAGE_KEY_LAST_USERNAME));
  const [isSubmitted, setIsSubmitted] = useState<boolean>(false);

  const onFinish = useCallback(
    async (creds: FormValues): Promise<void> => {
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
        user.isPasswordWeak = isPasswordWeak(creds.password || '');
        userStore.updateCurrentUser(user);
        if (user.isPasswordWeak) {
          handleWarning({
            level: ErrorLevel.Warn,
            publicMessage:
              'Your current password is either blank or weak according to current security recommendations. Please change your password.',
            publicSubject: WEAK_PASSWORD_SUBJECT,
            silent: false,
            type: ErrorType.Input,
          });
        }
        if (rbacEnabled) {
          // Now that we have logged in user, fetch userAssignments and userRoles and place into store.
          permissionStore.fetch(canceler.signal);
        }
        storage.set(STORAGE_KEY_LAST_USERNAME, creds.username);
      } catch (e) {
        const isBadCredentialsSync = isLoginFailure(e);
        setIsBadCredentials(isBadCredentialsSync); // this is not a sync operation
        uiActions.hideSpinner();
        if (isBadCredentialsSync) storage.remove(STORAGE_KEY_LAST_USERNAME);
        handleError(e, {
          isUserTriggered: true,
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

  const onValuesChange = useCallback((_changes: FormValues, values: FormValues): void => {
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
        <Input
          autoFocus
          data-testid={USERNAME_ID}
          placeholder="username"
          prefix={<Icon name="user" size="small" title="Username" />}
        />
      </Form.Item>
      <Form.Item name="password">
        <Input.Password
          data-testid={PASSWORD_ID}
          placeholder="password"
          prefix={<Icon name="lock" size="small" title="Password" />}
        />
      </Form.Item>
      {isBadCredentials && (
        <p className={[css.errorMessage, css.message].join(' ')}>Incorrect username or password.</p>
      )}
      <Form.Item>
        <Button
          data-testid={SUBMIT_ID}
          disabled={!canSubmit}
          htmlType="submit"
          loading={isSubmitted}
          type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>
  );

  return (
    <div className={css.base} data-test-component="detAuth">
      {loginForm}
      <p className={css.message}>
        Forgot your password, or need to manage users? Check out our&nbsp;
        <Link data-testid="docs" external path={paths.docs('/sysadmin-basics/users.html')} popout>
          docs
        </Link>
      </p>
    </div>
  );
};

export default DeterminedAuth;
