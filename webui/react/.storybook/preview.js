import 'styles/index.scss';
import 'styles/storybook.scss';

import { addDecorator } from "@storybook/react"
import 'loki/configure-react';

import ThemeDecorator from "storybook/ThemeDecorator"

addDecorator(ThemeDecorator);
