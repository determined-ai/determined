resource "aws_s3_bucket" "docs" {
  bucket = local.config.s3.bucket
}

# Set bucket object ownership if possible
resource "aws_s3_bucket_ownership_controls" "docs" {
  bucket = aws_s3_bucket.docs.id
  rule {
    object_ownership = "BucketOwnerPreferred"
  }
}

# disable the public access prevention controls AWS uses
resource "aws_s3_bucket_public_access_block" "docs" {
  bucket = aws_s3_bucket.docs.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_acl" "docs" {
  depends_on = [
    aws_s3_bucket_ownership_controls.docs,
    aws_s3_bucket_public_access_block.docs,
  ]

  bucket = aws_s3_bucket.docs.id
  acl    = "public-read"
}


resource "aws_s3_bucket_policy" "docs_policy" {
  bucket = aws_s3_bucket.docs.id
  policy = data.aws_iam_policy_document.docs.json
}

data "aws_iam_policy_document" "docs" {
  statement {
    sid    = "AddPerm"
    effect = "Allow"
    principals {
      type        = "AWS"
      identifiers = ["*"]
    }
    actions = [
      "s3:GetObject",
    ]
    resources = [
      "${aws_s3_bucket.docs.arn}/*",
    ]
  }
}

resource "aws_s3_bucket_website_configuration" "docs" {
  bucket = aws_s3_bucket.docs.id

  index_document {
    suffix = "index.html"
  }

  error_document {
    key = "/latest/404.html"
  }

  # example internal redirect
  #routing_rule {
  #  condition {
  #    key_prefix_equals = "docs/"
  #  }
  #  redirect {
  #    replace_key_prefix_with = "documents/"
  #  }
  #}
}

resource "aws_s3_object" "index" {
  bucket           = aws_s3_bucket.docs.id
  key              = "index.html"
  content          = "redirect to latest"
  content_type     = "text/html"
  website_redirect = "/latest/"
}

resource "aws_s3_object" "robots" {
  bucket       = aws_s3_bucket.docs.id
  key          = "robots.txt"
  content      = local.config.s3.robots.content
  content_type = "text"
}

# TODO: replace deprecated null_resource with terraform_data
# https://developer.hashicorp.com/terraform/language/resources/terraform-data
resource "null_resource" "upload" {
  triggers = {
    version = "${var.det_version}"
  }

  provisioner "local-exec" {
    command = <<-EOT
      aws s3 sync ../site/html s3://${aws_s3_bucket.docs.id}/${var.det_version} ;
      aws cloudfront create-invalidation --distribution-id ${aws_cloudfront_distribution.distribution.id} --paths '/${var.det_version}/*'
    EOT
  }
}

resource "null_resource" "upload_latest" {
  triggers = {
    version = "${var.det_version}"
  }

  provisioner "local-exec" {
    command = <<-EOT
      aws s3 sync ../site/html s3://${aws_s3_bucket.docs.id}/latest --delete ;
      aws cloudfront create-invalidation --distribution-id ${aws_cloudfront_distribution.distribution.id} --paths '/latest/*'
    EOT
  }
}
