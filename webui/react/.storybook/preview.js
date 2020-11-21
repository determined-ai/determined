import 'styles/index.scss';
import 'styles/storybook.scss';

import { addDecorator } from "@storybook/react"

import ThemeDecorator from "storybook/ThemeDecorator"

addDecorator(ThemeDecorator);

export const parameters = { layout: 'centered' };
