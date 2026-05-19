variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-east-1"
}

variable "owner" {
  description = "Owner tag applied to every resource."
  type        = string
  default     = "edulms-sre-student"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.20.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for the two public subnets."
  type        = list(string)
  default     = ["10.20.1.0/24", "10.20.2.0/24"]
}

variable "availability_zones" {
  description = "Two AZs in the chosen region."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

variable "instance_type" {
  description = "EC2 instance type for cluster nodes."
  type        = string
  default     = "t3.medium"
}

variable "node_count" {
  description = "Number of EC2 nodes that will run Docker / k8s."
  type        = number
  default     = 2
}

variable "ssh_key_name" {
  description = "Existing EC2 KeyPair name used for SSH access."
  type        = string
  default     = "edulms-sre-key"
}

variable "allowed_ssh_cidr" {
  description = "CIDR allowed to SSH into the nodes."
  type        = string
  default     = "0.0.0.0/0"
}
