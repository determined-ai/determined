import React, { useEffect } from 'react';

import useSettings from 'hooks/useSettings';
import useTheme from 'hooks/useTheme';
import { DarkLight } from 'themes';

import css from './ThemeToggle.module.scss';
import { config, Settings, Mode } from './ThemeToggle.settings';

interface ThemeOption {
  className: Mode;
  displayName: string;
  next: Mode
}

const ThemeOptions: {[theme: string] : ThemeOption} = {
  [Mode.LIGHT]: {
    className: Mode.LIGHT,
    displayName: 'Light Mode',
    next: Mode.DARK,
  },
  [Mode.DARK]: {
    className: Mode.DARK,
    displayName: 'Dark Mode',
    next: Mode.SYSTEM,
  },
  [Mode.SYSTEM]: {
    className: Mode.SYSTEM,
    displayName: 'System Mode',
    next: Mode.LIGHT,
  },
};

const ThemeToggle: React.FC = () => {

  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(config);

  const { setMode, themeMode } = useTheme();

  const theme = ThemeOptions[settings.theme];
  const classes = [ css.toggler ];
  classes.push(css[theme.className]);

  const changeTheme = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    const newThemeOption = theme.next;
    updateSettings({ theme: newThemeOption });
    setMode(newThemeOption);
  };

  useEffect(() => {
    /**
     * Ensure that the UI is loaded in Dark Mode if the user has
     * chosen it as their theme.
     */
    if(themeMode !== DarkLight.Dark && settings.theme === Mode.DARK){
      setMode(Mode.DARK);
    }
  });

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
