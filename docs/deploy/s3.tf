resource "aws_s3_bucket" "docs" {
  bucket = "hpe-mlde-docs"
  acl    = "public-read"

  website {
    index_document = "index.html"
    error_document = "/latest/404.html"
  }
}

resource "aws_s3_bucket_policy" "docs_policy" {
  bucket = "${aws_s3_bucket.docs.id}"

  policy = <<POLICY
{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AddPerm",
      "Effect":"Allow",
      "Principal": "*",
      "Action":["s3:GetObject"],
      "Resource":["${aws_s3_bucket.docs.arn}/*"]
    }
  ]
}
POLICY
}

resource "aws_s3_bucket_object" "index" {
  bucket           = "${aws_s3_bucket.docs.id}"
  key              = "index.html"
  content          = "redirect to latest"
  content_type     = "text/html"
  website_redirect = "/latest/"
}

resource "aws_s3_bucket_object" "robots" {
  bucket           = "${aws_s3_bucket.docs.id}"
  key              = "robots.txt"
  content          = "User-agent: *\nSitemap: https://hpe-mlde.determined.ai/latest/sitemap.xml"
  content_type     = "text"
}

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
