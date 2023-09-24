terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }

  required_version = ">= 1.2.0"
  backend "s3" {
    bucket = "acnil-terraform"
    key    = "tf"
    region = "eu-west-1"
  }
}

provider "aws" {
  region = "eu-west-1"
}

variable "bot_token" {
  description = "telegram bot token"
  type        = string
  sensitive   = true
}

variable "sheet_id" {
  description = "sheet used for inventory"
  type        = string
  sensitive   = true
}

variable "audit_sheet_id" {
  description = "sheet used for audit purposes"
  type        = string
  sensitive   = true
}

output "function_url" {
  value = module.bot_handler.lambda_function_url
}

variable "sheets_private_key" {
  description = "private key to access google sheets"
  type        = string
  sensitive   = true
}

variable "sheets_private_key_id" {
  description = "private key id to access google sheets"
  type        = string
  sensitive   = true
}

variable "sheets_email" {
  description = "Email used to interact with google sheets"
  type        = string
  sensitive   = false
}

//https://github.com/terraform-aws-modules/terraform-aws-lambda/tree/v6.0.0
module "bot_handler" {
  source = "terraform-aws-modules/lambda/aws"

  function_name              = "acnil-bot"
  description                = "Function to control acnil bot"
  handler                    = "bootstrap"
  runtime                    = "provided.al2"
  create_lambda_function_url = true
  architectures              = ["x86_64"]
  memory_size                = "128"
  timeout                    = "3"
  source_path                = "../cmd/lambda/package"

  environment_variables = {
    AUDIT_SHEET_ID : var.audit_sheet_id,
    SHEET_ID : var.sheet_id,
    TOKEN : var.bot_token,
    SHEETS_PRIVATE_KEY_ID : var.sheets_private_key_id
    SHEETS_PRIVATE_KEY : var.sheets_private_key
    SHEETS_EMAIL : var.sheets_email
  }
  cloudwatch_logs_retention_in_days = 14
}


module "audit_handler" {
  source = "terraform-aws-modules/lambda/aws"

  function_name = "audit-acnil-bot"
  description   = "Function to monitor audit"
  handler       = "bootstrap"
  runtime       = "provided.al2"
  architectures = ["x86_64"]
  memory_size   = "128"
  timeout       = "5"
  source_path   = "../cmd/auditLambda/package"

  environment_variables = {
    AUDIT_SHEET_ID : var.audit_sheet_id,
    SHEET_ID : var.sheet_id,
    TOKEN : var.bot_token,
    SHEETS_PRIVATE_KEY_ID : var.sheets_private_key_id
    SHEETS_PRIVATE_KEY : var.sheets_private_key
    SHEETS_EMAIL : var.sheets_email
  }
  cloudwatch_logs_retention_in_days = 14

  create_current_version_allowed_triggers = false
  allowed_triggers = {
    ScanAmiRule = {
      principal  = "events.amazonaws.com"
      source_arn = module.eventbridge.eventbridge_rule_arns["crons"]
    }
  }
}

# // https://registry.terraform.io/modules/terraform-aws-modules/eventbridge/aws/latest
module "eventbridge" {
  source = "terraform-aws-modules/eventbridge/aws"

  create_bus = false

  rules = {
    crons = {
      description = "Trigger for a audit acnil-bot Lambda"
      ## 1m
      # schedule_expression = "cron(0/1 * * * ? *)"
      ## every day at midnight
      schedule_expression = "cron(0 0 * * ? *)"
    }
  }

  targets = {
    crons = [
      {
        name  = module.audit_handler.lambda_function_name
        arn   = module.audit_handler.lambda_function_arn
        input = jsonencode({ "job" : "DoIT" })
      }
    ]
  }
}
