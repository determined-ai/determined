import { CustomCell, GridCellKind, Theme } from '@hpe.com/glide-data-grid';
import {
  BaseDrawArgs,
  CustomRenderer,
} from '@hpe.com/glide-data-grid/dist/ts/data-grid/cells/cell-types';

import { roundedRect } from '../utils';

function drawCheckbox(
  ctx: CanvasRenderingContext2D,
  theme: Theme,
  checked: boolean,
  x: number,
  y: number,
  width: number,
  height: number,
  highlighted: boolean,
  hoverX = -20,
  hoverY = -20,
) {
  const centerX = x + 0.5 * width;
  const centerY = y + 0.5 * height;

  const checkBoxWidth = 0.4 * height; // checkbox width proportional to cell height
  const emptyCheckBoxWidth = 0.375 * height;

  const hoverHelper = 0.29 * height;
  const hovered =
    Math.abs(hoverX - 0.5 * width) < hoverHelper && Math.abs(hoverY - 0.5 * height) < hoverHelper;

  const rectBordRadius = 3;
  const posHelper = 0.19 * height;

  if (checked) {
    ctx.beginPath();
    roundedRect(
      ctx,
      centerX - 0.5 * checkBoxWidth,
      centerY - 0.5 * checkBoxWidth,
      checkBoxWidth,
      checkBoxWidth,
      rectBordRadius,
    );

    ctx.fillStyle = highlighted ? theme.accentColor : theme.textMedium;
    ctx.fill();

    ctx.beginPath();

    const r = height;
    const cx = centerX - posHelper + 0.2 * r;
    const cy = centerY - posHelper + 0.2 * r;
    ctx.moveTo(cx - 0.07 * r, cy - 0.01 * r);
    ctx.lineTo(cx - 0.02 * r, cy + 0.04 * r);
    ctx.lineTo(cx + 0.06 * r, cy - 0.05 * r);

    ctx.strokeStyle = theme.bgCell;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';
    ctx.lineWidth = 1.5;
    ctx.stroke();
  } else {
    ctx.beginPath();
    roundedRect(
      ctx,
      centerX - posHelper,
      centerY - posHelper,
      emptyCheckBoxWidth,
      emptyCheckBoxWidth,
      rectBordRadius,
    );

    ctx.lineWidth = 1;
    ctx.strokeStyle = hovered ? theme.textDark : theme.textMedium;
    ctx.stroke();
  }
}

function drawBoolean(args: BaseDrawArgs, checked: boolean) {
  const { ctx, theme, rect, highlighted, hoverX, hoverY } = args;
  const { x, y, width: w, height: h } = rect;

  drawCheckbox(ctx, theme, checked, x, y, w, h, highlighted, hoverX, hoverY);

  if (checked) {
    ctx.beginPath();
    ctx.rect(x, y, 3, h);
    ctx.fill();
  }
}

type CheckboxCellProps = {
  kind: 'checkbox-cell';
  checked: boolean;
};
export type CheckboxCell = CustomCell<CheckboxCellProps>;

const renderer: CustomRenderer<CheckboxCell> = {
  draw: (a, cell) => drawBoolean(a, cell.data.checked),
  isMatch: (c): c is CheckboxCell => (c.data as CheckboxCellProps).kind === 'checkbox-cell',
  kind: GridCellKind.Custom,
  measure: () => 40,
  needsHover: true,
  needsHoverPosition: true,
};

export default renderer;
