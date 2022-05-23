import React, {useState} from 'react';

import { BaseType, SettingsConfig } from 'hooks/useSettings';
import useSettings from 'hooks/useSettings';
import css from './ThemeToggle.module.scss';
import { string } from 'fp-ts';
import { getDisplayName } from 'utils/user';

interface Settings {
    theme: string;
}

enum ThemeClass {
    SYSTEM = 'system',
    LIGHT = 'light',
    DARK = 'dark'
}

const settingsConfig: SettingsConfig = {
    settings: [
      {
        defaultValue: ThemeClass.SYSTEM,
        key: 'theme',
        storageKey: 'theme',
        type: { baseType: BaseType.String },
      },
    ],
    storagePath: 'settings/theme',
  };

  interface ThemeOption {
    className: ThemeClass;
    displayName: string;
    next: ThemeClass
}

const ThemeOptions: {[theme: string] : ThemeOption} = {
        [ThemeClass.LIGHT] : {
            displayName: 'Light Mode',
            next: ThemeClass.DARK,
            className: ThemeClass.LIGHT
        }, 
        [ThemeClass.DARK] : {
            displayName: 'Dark Mode',
            next: ThemeClass.SYSTEM,
            className: ThemeClass.DARK
        }, 
        [ThemeClass.SYSTEM] : {
            displayName: 'System Mode',
            next: ThemeClass.LIGHT,
            className: ThemeClass.SYSTEM
        }, 
}

const ThemeToggle: React.FC = () => {

    const {
        settings,
        updateSettings,
      } = useSettings<Settings>(settingsConfig);

    let theme = ThemeOptions[settings.theme];
    const classes=[css.toggler];
    classes.push(css[theme.className])
    
    const changeTheme = (e: React.MouseEvent) => {
        e.stopPropagation(); 
        e.preventDefault();
        updateSettings({theme: theme.next});
    }

return (
    <div className={css.base} onClick={changeTheme}>
        <div className={css.container}>
            <div className={classes.join(' ')}>
            </div>
            <div className={css.mode}>
                {theme.displayName}
            </div>
        </div>
    </div>
  )
}

export default ThemeToggle;


