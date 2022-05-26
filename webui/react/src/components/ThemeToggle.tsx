import React, { useEffect } from 'react';

import useSettings from 'hooks/useSettings';
import useTheme from 'hooks/useTheme';
import { DarkLight } from 'themes';

import css from './ThemeToggle.module.scss';
import { config, Settings, ThemeClass } from './ThemeToggle.settings';

interface ThemeOption {
  className: ThemeClass;
  displayName: string;
  next: ThemeClass
}

const ThemeOptions: {[theme: string] : ThemeOption} = {
  [ThemeClass.LIGHT]: {
    className: ThemeClass.LIGHT,
    displayName: 'Light Mode',
    next: ThemeClass.DARK,
  },
  [ThemeClass.DARK]: {
    className: ThemeClass.DARK,
    displayName: 'Dark Mode',
    next: ThemeClass.SYSTEM,
  },
  [ThemeClass.SYSTEM]: {
    className: ThemeClass.SYSTEM,
    displayName: 'System Mode',
    next: ThemeClass.LIGHT,
  },
};

const ThemeToggle: React.FC = () => {

  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(config);

  const { setMode, mode, systemMode } = useTheme();

  const theme = ThemeOptions[settings.theme];
  const classes = [ css.toggler ];
  classes.push(css[theme.className]);

  const changeTheme = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    const newThemeOption = theme.next;
    updateSettings({ theme: newThemeOption });
    setMode(newThemeOption === ThemeClass.DARK ? DarkLight.Dark : DarkLight.Light);
  };

  useEffect(() => {
    /**
     * Ensure that the UI is loaded in Dark Mode if the user has
     * chosen it as their theme.
     */
    if(mode !== DarkLight.Dark && settings.theme === ThemeClass.DARK){
      setMode(DarkLight.Dark);
    }
  });

  useEffect(() => {
    /**
     * Ensure that the chosen UI theme is used when the OS or
     * browser appearance settings are updated.
     */
    if(systemMode === DarkLight.Dark && settings.theme === ThemeClass.DARK){
      setMode(DarkLight.Dark);
    } else if(systemMode === DarkLight.Light && settings.theme === ThemeClass.LIGHT){
      setMode(DarkLight.Light);
    } else if(systemMode === DarkLight.Light && settings.theme === ThemeClass.SYSTEM){
      setMode(DarkLight.Light);
    }
  }, [ systemMode, setMode, settings.theme ]);

  return (
    <div className={css.base} onClick={changeTheme}>
      <div className={css.container}>
        <div className={classes.join(' ')} />
        <div className={css.mode}>
          {theme.displayName}
        </div>
      </div>
    </div>
  );
};

export default ThemeToggle;
