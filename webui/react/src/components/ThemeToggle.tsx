import React from 'react';

import css from 'components/ThemeToggle.module.scss';
import useUI from 'stores/contexts/UI';
import { Mode } from 'utils/themes';

interface Props {
  iconOnly?: boolean;
}

interface ThemeOption {
  className: Mode;
  displayName: string;
  next: Mode;
}

export const ThemeOptions: Record<Mode, ThemeOption> = {
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

const ThemeToggle: React.FC<Props> = ({ iconOnly = false }) => {
  const {
    ui: { mode: uiMode },
    actions: { setMode },
  } = useUI();

  const classes = [css.base];
  if (iconOnly) classes.push(css.iconOnly);

  const togglerClasses = [css.toggler];
  const currentThemeOption = ThemeOptions[uiMode];
  togglerClasses.push(css[currentThemeOption.className]);

  const newThemeMode = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setMode(currentThemeOption.next);
  };

  return (
    <div className={classes.join(' ')} onClick={newThemeMode}>
      <div className={css.container}>
        <div className={togglerClasses.join(' ')} />
        <div className={css.mode}>{currentThemeOption.displayName}</div>
      </div>
    </div>
  );
};

export default ThemeToggle;
