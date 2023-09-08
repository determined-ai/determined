import {
  blend,
  CustomCell,
  CustomRenderer,
  getMiddleCenterBias,
  GridCellKind,
  measureTextCached,
} from '@hpe.com/glide-data-grid';

import { roundedRect } from 'pages/F_ExpList/glide-table/custom-renderers/utils';

interface LinkCellProps {
  readonly kind: 'link-cell';
  /**
   * Used to hand tune the position of the underline as this is not a native canvas capability, it can need tweaking
   * for different fonts.
   */
  readonly underlineOffset?: number;
  readonly maxLinks?: number;
  readonly navigateOn?: 'click' | 'control-click';
  readonly link: {
    readonly title: string;
    readonly href: string;
    readonly unmanaged?: boolean;
  };
}

export type LinkCell = CustomCell<LinkCellProps>;

const TAG_HEIGHT = 20;
const TAG_CONTENT = 'Unmanaged';

function onClickSelect(e: Parameters<NonNullable<CustomRenderer<LinkCell>['onSelect']>>[0]) {
  const useCtrl = e.cell.data.navigateOn !== 'click';
  if (useCtrl !== e.ctrlKey) return undefined;
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d', { alpha: false });
  if (ctx === null) return;

  const { posX: hoverX, bounds: rect, cell, theme } = e;
  const font = `${theme.baseFontStyle} ${theme.fontFamily}`;
  ctx.font = font;

  const { link } = cell.data;

  const rectHoverX = rect.x + hoverX;

  const isHovered = rectHoverX > rect.x && rectHoverX < rect.x + rect.width;

  if (isHovered) {
    return link;
  }

  return undefined;
}

const renderer: CustomRenderer<LinkCell> = {
  draw: (args, cell) => {
    const { ctx, rect, theme, hoverX = -100, highlighted } = args;
    const { link, underlineOffset = 5 } = cell.data;
    if (link === undefined) return;

    const xPad = theme.cellHorizontalPadding;

    let drawX = rect.x + xPad;

    const rectHoverX = rect.x + hoverX;

    const font = `${theme.baseFontStyle} ${theme.fontFamily}`;

    const middleCenterBias = getMiddleCenterBias(ctx, font);
    const drawY = rect.y + rect.height / 2 + middleCenterBias;

    const metrics = measureTextCached(link.title, ctx, font);
    const commaMetrics = metrics;

    const isHovered = rectHoverX > rect.x && rectHoverX < rect.x + rect.width;

    if (isHovered) {
      ctx.moveTo(drawX, Math.floor(drawY + underlineOffset) + 0.5);
      ctx.lineTo(drawX + metrics.width, Math.floor(drawY + underlineOffset) + 0.5);

      // ctx.lineWidth = 1;
      ctx.strokeStyle = theme.linkColor;
      ctx.stroke();

      ctx.fillStyle = highlighted ? blend(theme.accentLight, theme.bgCell) : theme.bgCell;
      ctx.fillText(link.title, drawX - 1, drawY);
      ctx.fillText(link.title, drawX + 1, drawY);

      ctx.fillText(link.title, drawX - 2, drawY);
      ctx.fillText(link.title, drawX + 2, drawY);
    }
    ctx.fillStyle = theme.linkColor;
    ctx.fillText(link.title, drawX, drawY);
    if (link.unmanaged) {
      const x = drawX + commaMetrics.width + 8;
      const y = drawY - TAG_HEIGHT / 2;
      ctx.fillStyle = '#132231';
      ctx.lineWidth = 2;
      ctx.strokeStyle = theme.textBubble;
      ctx.beginPath();
      roundedRect(ctx, x, y, measureTextCached(TAG_CONTENT, ctx).width + 8, TAG_HEIGHT, 4);
      ctx.stroke();
      ctx.fill();
      ctx.fillStyle = '#fff';
      ctx.fillText(
        TAG_CONTENT,
        x + 4,
        y + TAG_HEIGHT / 2 + getMiddleCenterBias(ctx, `12px ${theme.fontFamily}`),
      );
    }

    drawX += commaMetrics.width + 4;

    return true;
  },
  isMatch: (c): c is LinkCell => (c.data as LinkCellProps).kind === 'link-cell',
  kind: GridCellKind.Custom,
  measure: (ctx, cell, theme) => {
    const { link } = cell.data;
    if (link === undefined) return 0;

    return ctx.measureText(link.title).width + theme.cellHorizontalPadding * 2;
  },
  needsHover: true,
  needsHoverPosition: true,
  onSelect: (e) => {
    if (onClickSelect(e) !== undefined) {
      e.preventDefault();
    }
  },
  provideEditor: () => undefined,
};

export default renderer;
