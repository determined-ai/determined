import React, { useEffect } from 'react';

import useTheme from 'hooks/useTheme';
import {Mode} from 'hooks/useTheme.settings';

import css from './ThemeToggle.module.scss';

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

  const { mode,  updateTheme } = useTheme();

  const classes = [ css.toggler ];
  const currentThemeOption = ThemeOptions[mode];
  classes.push(css[currentThemeOption.className]);

  const changeTheme = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    const newThemeOption = currentThemeOption.next;
    updateTheme(newThemeOption);
  };

  return (
    <div className={css.base} onClick={changeTheme}>
      <div className={css.container}>
        <div className={classes.join(' ')} />
        <div className={css.mode}>
          {currentThemeOption.displayName}
        </div>
      </div>
    </div>
  );
};

export default ThemeToggle;
