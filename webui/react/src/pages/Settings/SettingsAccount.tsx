import { CheckOutlined, CloseOutlined, EditOutlined } from '@ant-design/icons';
import { Divider, Typography } from 'antd';
import { useObservable } from 'micro-observables';
import React, { useCallback, useState } from 'react';

import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import Avatar from 'components/kit/UserAvatar';
import useModalPasswordChange from 'hooks/useModal/UserSettings/useModalPasswordChange';
import { patchUser } from 'services/api';
import { Size } from 'shared/components/Avatar';
import { ErrorType } from 'shared/utils/error';
import usersStore from 'stores/usersObserve';
import { message } from 'utils/dialogApi';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

import css from './SettingsAccount.module.scss';

interface FormUsernameInputs {
  username: string;
}

interface FormDisplaynameInputs {
  displayName: string;
}

export const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
export const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
export const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';
export const CHANGE_PASSWORD_TEXT = 'Change Password';

const SettingsAccount: React.FC = () => {
  const [usernameForm] = Form.useForm<FormUsernameInputs>();
  const [displaynameForm] = Form.useForm<FormDisplaynameInputs>();
  const loadableCurrentUser = useObservable(usersStore.getCurrentUser());
  const currentUser = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const [isUsernameEditable, setIsUsernameEditable] = useState<boolean>(false);
  const [isDisplaynameEditable, setIsDisplaynameEditable] = useState<boolean>(false);

  const { contextHolder: modalPasswordChangeContextHolder, modalOpen: openChangePasswordModal } =
    useModalPasswordChange();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [openChangePasswordModal]);

  const handleSaveDisplayName = useCallback(async (): Promise<void | Error> => {
    const values = await displaynameForm.validateFields();
    try {
      const user = await patchUser({
        userId: currentUser?.id || 0,
        userParams: { displayName: values.displayName },
      });
      usersStore.updateUsers(user);
      message.success(API_DISPLAYNAME_SUCCESS_MESSAGE);
      setIsDisplaynameEditable(false);
    } catch (e) {
      handleError(e, { silent: false, type: ErrorType.Input });
      return e as Error;
    }
  }, [currentUser?.id, displaynameForm]);

  const handleSaveUsername = useCallback(async (): Promise<void | Error> => {
    const values = await usernameForm.validateFields();
    try {
      const user = await patchUser({
        userId: currentUser?.id || 0,
        userParams: { username: values.username },
      });
      usersStore.updateUsers(user);
      message.success(API_USERNAME_SUCCESS_MESSAGE);
      setIsUsernameEditable(false);
    } catch (e) {
      message.error(API_USERNAME_ERROR_MESSAGE);
      handleError(e, { silent: true, type: ErrorType.Input });
      return e as Error;
    }
  }, [currentUser?.id, usernameForm]);

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip size={Size.ExtraLarge} user={currentUser} />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Username</label>
        {!isUsernameEditable ? (
          <div className={css.displayInfo}>
            <span>{currentUser?.username ?? ''}</span>
            <Button
              data-testid="edit-username"
              icon={<EditOutlined />}
              onClick={() => setIsUsernameEditable(true)}
            />
          </div>
        ) : (
          <Form
            className={css.form}
            form={usernameForm}
            layout="inline"
            onFinish={handleSaveUsername}>
            <Form.Item
              initialValue={currentUser?.username ?? ''}
              name="username"
              noStyle
              rules={[{ message: 'Please input your username', required: true }]}>
              <Input maxLength={32} placeholder="Add username" style={{ widows: '80%' }} />
            </Form.Item>
            <Form.Item noStyle>
              <Button htmlType="submit" icon={<CheckOutlined />} type="primary" />
            </Form.Item>
            <Form.Item noStyle>
              <Button icon={<CloseOutlined />} onClick={() => setIsUsernameEditable(false)} />
            </Form.Item>
          </Form>
        )}
      </div>
      <Divider />
      <div className={css.row}>
        <label>Display Name</label>
        {!isDisplaynameEditable ? (
          <div className={css.displayInfo}>
            <span>
              {currentUser?.displayName || <Typography.Text disabled>N/A</Typography.Text>}
            </span>
            <Button
              data-testid="edit-displayname"
              icon={<EditOutlined />}
              onClick={() => setIsDisplaynameEditable(true)}
            />
          </div>
        ) : (
          <Form
            className={css.form}
            form={displaynameForm}
            layout="inline"
            onFinish={handleSaveDisplayName}>
            <Form.Item initialValue={currentUser?.displayName ?? ''} name="displayName" noStyle>
              <Input maxLength={32} placeholder="Add display name" style={{ widows: '80%' }} />
            </Form.Item>
            <Form.Item noStyle>
              <Button htmlType="submit" icon={<CheckOutlined />} type="primary" />
            </Form.Item>
            <Form.Item noStyle>
              <Button icon={<CloseOutlined />} onClick={() => setIsDisplaynameEditable(false)} />
            </Form.Item>
          </Form>
        )}
      </div>
      <Divider />
      <div className={css.row}>
        <label>Password</label>
        <Button onClick={handlePasswordClick}>{CHANGE_PASSWORD_TEXT}</Button>
      </div>
      {modalPasswordChangeContextHolder}
    </div>
  );
};

export default SettingsAccount;
