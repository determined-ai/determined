import { Divider } from 'antd';
import React, { useCallback } from 'react';

import Button from 'components/kit/Button';
import { Column, Columns } from 'components/kit/Columns';
import Drawer from 'components/kit/Drawer';
import InlineForm from 'components/kit/InlineForm';
import Input from 'components/kit/Input';
import InputShortcut from 'components/kit/InputShortcut';
import { useModal } from 'components/kit/Modal';
import PasswordChangeModalComponent from 'components/PasswordChangeModal';
import Section from 'components/Section';
import { useSettings } from 'hooks/useSettings';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'pages/Settings/UserSettings.settings';
import { patchUser } from 'services/api';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './SettingsAccount.module.scss';

export const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
export const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
export const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';
export const CHANGE_PASSWORD_TEXT = 'Change Password';

interface Props {
  show: boolean;
  onClose: () => void;
}

const SettingsAccount: React.FC<Props> = ({ show, onClose }: Props) => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const info = useObservable(determinedStore.info);

  const PasswordChangeModal = useModal(PasswordChangeModalComponent);
  const {
    settings: {
      jupyterLab: jupyterLabShortcut,
      omnibar: omnibarShortcut,
      navbarCollapsed: navbarCollapsedShortcut,
    },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig);

  const handleSaveDisplayName = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: currentUser?.id || 0,
          userParams: { displayName: newValue as string },
        });
        userStore.updateUsers(user);
        message.success(API_DISPLAYNAME_SUCCESS_MESSAGE);
      } catch (e) {
        handleError(e, { silent: false, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser?.id],
  );

  const handleSaveUsername = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: currentUser?.id || 0,
          userParams: { username: newValue as string },
        });
        userStore.updateUsers(user);
        message.success(API_USERNAME_SUCCESS_MESSAGE);
      } catch (e) {
        message.error(API_USERNAME_ERROR_MESSAGE);
        handleError(e, { silent: true, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser?.id],
  );

  return (
    <Drawer open={show} placement="left" title="Settings" onClose={onClose}>
      <Section title="Profile">
        <div className={css.section}>
          <InlineForm<string>
            initialValue={currentUser?.username ?? ''}
            label="Username"
            required
            rules={[{ message: 'Please input your username', required: true }]}
            testId="username"
            onSubmit={handleSaveUsername}>
            <Input maxLength={32} placeholder="Add username" />
          </InlineForm>
          <InlineForm<string>
            initialValue={currentUser?.displayName ?? ''}
            label="Display Name"
            testId="displayname"
            onSubmit={handleSaveDisplayName}>
            <Input maxLength={32} placeholder="Add display name" />
          </InlineForm>
          {info.userManagementEnabled && (
            <>
              <Columns>
                <Column>
                  <label>Password</label>
                </Column>
                <Button onClick={PasswordChangeModal.open}>{CHANGE_PASSWORD_TEXT}</Button>
              </Columns>
              <PasswordChangeModal.Component />
            </>
          )}
        </div>
        <Divider />
      </Section>
      <Section title="Shortcuts">
        <div className={css.section}>
          <Columns>
            <Column>
              <label>Open Omnibar</label>
            </Column>
            <InputShortcut value={omnibarShortcut} />
          </Columns>
          <Columns>
            <Column>
              <label>Launch JupyterLab Notebook</label>
            </Column>
            <InputShortcut value={jupyterLabShortcut} />
          </Columns>
          <Columns>
            <Column>
              <label>Toggle Sidebar</label>
            </Column>
            <InputShortcut value={navbarCollapsedShortcut} />
          </Columns>
        </div>
      </Section>
    </Drawer>
  );
};

export default SettingsAccount;
