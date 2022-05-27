import React from 'react';

import useTheme from 'hooks/useTheme';
import { Mode } from 'hooks/useTheme.settings';

import css from './ThemeToggle.module.scss';

interface ThemeOption {
  className: Mode;
  displayName: string;
  next: Mode
}

const ThemeOptions: Record<Mode, ThemeOption> = {
  [Mode.Light]: {
    className: Mode.Light,
    displayName: 'Light Mode',
    next: Mode.Dark,

  },
  [Mode.Dark]: {
    className: Mode.Dark,
    displayName: 'Dark Mode',
    next: Mode.System,
  },
  [Mode.System]: {
    className: Mode.System,
    displayName: 'System Mode',
    next: Mode.Light,
  },
};

const ThemeToggle: React.FC = () => {

  const { mode, updateTheme } = useTheme();

  const classes = [ css.toggler ];
  const currentThemeOption = ThemeOptions[mode];
  classes.push(css[currentThemeOption.className]);

  const newThemeMode = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    const newThemeOption = currentThemeOption.next;
    updateTheme(newThemeOption);
  };

  return (
    <div className={css.base} onClick={newThemeMode}>
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
