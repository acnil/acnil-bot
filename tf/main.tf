terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }

  required_version = ">= 1.2.0"
  backend "s3" {
    bucket = "metalblueberry"
    key    = "acnil"
    region = "eu-west-1"
  }
}

provider "aws" {
  region = "eu-west-1"
}


// This is the md5 of the bot tokens. Used to validate that you are using the right tokens
// Feel free to add more worspaces if needed
locals {
  bot_token_md5 = {
    "default" : "ac2312fb0515871b88408d233f7c562e"
    "production" : "0606b1eb24a5ea79d095b081a55ce690"
  }
}

variable "bot_token" {
  description = "telegram bot token"
  type        = string
  sensitive   = true
}

variable "webhook_secret_token" {
  description = "Token that telegram will use to ensure the calls come from telegram"
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

  // This precondition is used to make sure you are using the right bot token for the environment
  // Mainly to detect cases where you have the wrong environment set.
  precondition {
    condition     = md5(var.bot_token) == local.bot_token_md5[terraform.workspace]
    error_message = format("the selected token doesn't match the expected for the workspace. The hash is \"%s\" but expected \"%s\"", nonsensitive(md5(var.bot_token)), local.bot_token_md5[terraform.workspace])
  }
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

  function_name              = format("%s-acnil-bot", terraform.workspace)
  description                = "Function to control acnil bot"
  handler                    = "bootstrap"
  runtime                    = "provided.al2"
  create_lambda_function_url = true
  architectures              = ["x86_64"]
  memory_size                = "128"
  timeout                    = "3"

  create_package         = false
  local_existing_package = "../cmd/lambda/package.zip"

  environment_variables = {
    AUDIT_SHEET_ID : var.audit_sheet_id,
    SHEET_ID : var.sheet_id,
    TOKEN : var.bot_token,
    SHEETS_PRIVATE_KEY_ID : var.sheets_private_key_id
    SHEETS_PRIVATE_KEY : var.sheets_private_key
    SHEETS_EMAIL : var.sheets_email
    WEBHOOK_SECRET_TOKEN : var.webhook_secret_token
  }
  cloudwatch_logs_retention_in_days = 14
}


module "audit_handler" {
  source = "terraform-aws-modules/lambda/aws"

  function_name = format("%s-audit-acnil-bot", terraform.workspace)
  description   = "Function to monitor audit"
  handler       = "bootstrap"
  runtime       = "provided.al2"
  architectures = ["x86_64"]
  memory_size   = "128"
  timeout       = "5"

  create_package         = false
  local_existing_package = "../cmd/auditLambda/package.zip"

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
      source_arn = resource.aws_cloudwatch_event_rule.daily.arn
    }
  }
}

# // https://registry.terraform.io/modules/terraform-aws-modules/eventbridge/aws/latest

resource "aws_cloudwatch_event_rule" "daily" {
  name        = format("%s-acnil-bot-daily_rule", terraform.workspace)
  description = "trigger lambda daily"

  # schedule_expression = "rate(5 minutes)"
  schedule_expression = "cron(0 0 * * ? *)"
}

resource "aws_cloudwatch_event_target" "lambda_target" {
  rule      = aws_cloudwatch_event_rule.daily.name
  target_id = "SendToLambda"
  arn       = module.audit_handler.lambda_function_arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id = "AllowExecutionFromEventBridge"
  action       = "lambda:InvokeFunction"
  # function_name = aws_lambda_function.test_lambda.function_name
  function_name = module.audit_handler.lambda_function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.daily.arn
}
