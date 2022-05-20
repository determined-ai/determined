import { number } from 'fp-ts';
import React, {useState} from 'react';

import css from './ThemeToggle.module.scss';

enum ThemeClass {
    SYSTEM = 'system',
    LIGHT = 'light',
    DARK = 'dark'
}
interface ThemeOption {
    className: ThemeClass;
    displayName: string;
}

const themeOptions: ThemeOption[] = 
[
    {
        className: ThemeClass.LIGHT,
        displayName: 'Light Mode'
    },
    {
        className: ThemeClass.DARK,
        displayName: 'Dark Mode'
    },
    {
        className: ThemeClass.SYSTEM,
        displayName: 'System Mode'
    }
]
const ThemeToggle: React.FC = () => {

    const [themeOption, setThemeOption] = useState<number>(0);
    const classes=[css.toggler];
    classes.push(css[themeOptions[themeOption].className])
    
return (
    <div className={css.base}>
        <div className={css.container}>
            <div onClick={(e) => {
            e.stopPropagation(); 
            e.preventDefault();
            const newThemeOption = themeOption === themeOptions.length-1 ? 0 : themeOption+1
            setThemeOption(newThemeOption)
        }}
        className={classes.join(' ')}>
        </div>
        <div className={css.mode}>
        Light Mode
        </div>
    </div>
    </div>
  )
}

export default ThemeToggle;


