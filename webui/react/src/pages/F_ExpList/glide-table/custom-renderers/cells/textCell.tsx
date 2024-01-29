import {
  CustomCell,
  CustomRenderer,
  getMiddleCenterBias,
  GridCellKind,
} from '@glideapps/glide-data-grid';

import { drawTextWithEllipsis } from 'pages/F_ExpList/glide-table/custom-renderers/utils';

interface TextCellProps {
  readonly kind: 'text-cell';
}

export type TextCell = CustomCell<TextCellProps>;

const renderer: CustomRenderer<TextCell> = {
  draw: (args, cell) => {
    const { ctx, rect, theme } = args;
    // hoverX = -100, highlighted

    const xPad = theme.cellHorizontalPadding;
    const font = `${theme.baseFontStyle} ${theme.fontFamily}`;
    const middleCenterBias = getMiddleCenterBias(ctx, font);
    const x = rect.x + xPad;
    const y = rect.y + rect.height / 2 + middleCenterBias;
    const maxWidth = rect.width - 2 * theme.cellHorizontalPadding;

    ctx.fillStyle = theme.textHeader;
    drawTextWithEllipsis(ctx, cell.copyData, x, y, maxWidth);

    return true;
  },
  isMatch: (c): c is TextCell => (c.data as TextCellProps).kind === 'text-cell',
  kind: GridCellKind.Custom,
  needsHover: true,
  provideEditor: () => undefined,
};

export default renderer;
