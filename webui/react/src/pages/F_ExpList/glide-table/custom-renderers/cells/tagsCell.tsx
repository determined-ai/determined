import {
  CustomCell,
  CustomRenderer,
  getMiddleCenterBias,
  GridCellKind,
  measureTextCached,
  Rectangle,
} from '@hpe.com/glide-data-grid';

import { roundedRect } from '../utils';

interface TagsCellProps {
  readonly kind: 'tags-cell';
  readonly tags: readonly string[];
  readonly readonly?: boolean;
  readonly possibleTags: readonly {
    tag: string;
    color: string;
  }[];
}

export type TagsCell = CustomCell<TagsCellProps>;

const TAG_HEIGHT = 20;
const TAG_GAP = 8;
const TAG_INNER_PAD = 6;

const renderer: CustomRenderer<TagsCell> = {
  draw: (args, cell) => {
    const { ctx, theme, rect } = args;
    const { possibleTags, tags } = cell.data;

    const drawArea: Rectangle = {
      height: rect.height - 2 * theme.cellVerticalPadding,
      width: rect.width - 2 * theme.cellHorizontalPadding,
      x: rect.x + theme.cellHorizontalPadding,
      y: rect.y + theme.cellVerticalPadding,
    };
    const rows = Math.max(1, Math.floor(drawArea.height / (TAG_HEIGHT + TAG_INNER_PAD)));

    let x = drawArea.x;
    let row = 1;
    let y = drawArea.y + (drawArea.height - rows * TAG_HEIGHT - (rows - 1) * TAG_INNER_PAD) / 2;
    for (const tag of tags) {
      const color = possibleTags.find((t) => t.tag === tag)?.color ?? theme.bgBubble;

      ctx.font = `12px ${theme.fontFamily}`;
      const metrics = measureTextCached(tag, ctx);
      const width = metrics.width + TAG_INNER_PAD * 2;
      const textY = TAG_HEIGHT / 2;

      if (x !== drawArea.x && x + width > drawArea.x + drawArea.width && row < rows) {
        row++;
        y += TAG_HEIGHT + TAG_INNER_PAD;
        x = drawArea.x;
      }

      ctx.fillStyle = color;
      ctx.lineWidth = 2;
      ctx.strokeStyle = theme.textBubble;
      ctx.beginPath();
      roundedRect(ctx, x, y, width, TAG_HEIGHT, TAG_HEIGHT / 2);
      ctx.stroke();
      ctx.fill();

      ctx.fillStyle = theme.textDark;
      ctx.fillText(
        tag,
        x + TAG_INNER_PAD,
        y + textY + getMiddleCenterBias(ctx, `12px ${theme.fontFamily}`),
      );

      x += width + TAG_GAP;
      if (x > drawArea.x + drawArea.width && row >= rows) break;
    }

    return true;
  },
  isMatch: (c): c is TagsCell => (c.data as TagsCellProps).kind === 'tags-cell',
  kind: GridCellKind.Custom,
  measure: (ctx, cell, theme) => {
    const { tags } = cell.data;

    let tagsWidth = 0;
    for (const tag of tags) {
      tagsWidth += ctx.measureText(tag).width + 2 * TAG_INNER_PAD;
    }
    tagsWidth += Math.max(tags.length - 1, 0) * TAG_GAP;

    return tagsWidth + theme.cellHorizontalPadding * 2;
  },
  onPaste: (v, d) => ({
    ...d,
    tags: d.possibleTags
      .map((x) => x.tag)
      .filter((x) =>
        v
          .split(',')
          .map((s) => s.trim())
          .includes(x),
      ),
  }),
  provideEditor: () => undefined,
};

export default renderer;
