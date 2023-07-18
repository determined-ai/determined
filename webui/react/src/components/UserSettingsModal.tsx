import { Alert } from 'antd';
import { Map } from 'immutable';
import { useMemoizedObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import useUI from 'stores/contexts/UI';
import userSettings from 'stores/userSettings';
import { Json } from 'types';
import { isJsonObject, isObject } from 'utils/data';
import handleError from 'utils/error';
import { Loadable, Loaded } from 'utils/loadable';
import { Mode } from 'utils/themes';

import CodeEditor from './kit/CodeEditor';

interface Props {
  onSave?: () => void;
}

const UserSettingsModal: React.FC<Props> = ({ onSave }: Props) => {
  const { actions: uiActions } = useUI();
  const [configError, setConfigError] = useState(false);
  const initialSettingsString = useMemoizedObservable<Loadable<string>>(
    () =>
      userSettings
        .getAll()
        .select((loadableState) =>
          Loadable.map(loadableState, (state) => JSON.stringify(state, undefined, ' ')),
        ),
    [],
  );
  const [editedSettingsString, setEditedSettingsString] =
    useState<Loadable<string>>(initialSettingsString);

  const editedSettings: Map<string, Json> | undefined = useMemo(
    () =>
      Loadable.match(editedSettingsString, {
        Loaded: (settingsString) => {
          try {
            const obj = JSON.parse(settingsString);
            setConfigError(false);
            return isObject(obj) ? Map<string, Json>(obj) : undefined;
          } catch {
            setConfigError(true);
            return;
          }
        },
        NotLoaded: () => undefined,
      }),
    [editedSettingsString],
  );

  useEffect(() => {
    if (Loadable.isLoaded(editedSettingsString)) return;

    setEditedSettingsString(initialSettingsString);
  }, [editedSettingsString, initialSettingsString]);

  const handleSave = useCallback(() => {
    if (!editedSettings) return;

    onSave?.();
    userSettings.overwrite(editedSettings);

    // We have to special case the mode because otherwise ThemeProvider will revert the change.
    const theme = editedSettings.get('settings-theme');
    if (
      theme !== undefined &&
      isJsonObject(theme) &&
      Object.values(Mode).includes(theme.mode as Mode)
    ) {
      uiActions.setMode(theme.mode as Mode);
    }
  }, [editedSettings, onSave, uiActions]);

  const handleChange = useCallback((newSettings: string) => {
    setEditedSettingsString(Loaded(newSettings));
  }, []);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        disabled: editedSettings === undefined,
        handleError,
        handler: handleSave,
        text: 'Save Settings',
      }}
      title="Edit Raw Settings">
      <CodeEditor
        files={[
          {
            content: editedSettingsString,
            key: 'settings.json',
            title: 'settings.json',
          },
        ]}
        height="400px"
        onChange={handleChange}
        onError={handleError}
      />
      {configError && <Alert message="Invalid JSON" type="error" />}
    </Modal>
  );
};

export default UserSettingsModal;
