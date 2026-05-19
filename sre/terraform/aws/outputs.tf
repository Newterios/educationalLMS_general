output "vpc_id" {
  description = "ID of the created VPC."
  value       = aws_vpc.this.id
}

output "node_public_ips" {
  description = "Public IPv4 addresses of every EC2 node."
  value       = aws_instance.node[*].public_ip
}

output "node_private_ips" {
  description = "Private IPv4 addresses of every EC2 node."
  value       = aws_instance.node[*].private_ip
}

output "ansible_inventory_hint" {
  description = "Hint command to render an Ansible inventory from this output."
  value = <<-EOT
    Run the following to refresh ../../ansible/inventory.ini :

    cat > ../../ansible/inventory.ini <<INV
    [managers]
    ${aws_instance.node[0].public_ip} ansible_user=ubuntu

    [workers]
    %{ for ip in slice(aws_instance.node[*].public_ip, 1, length(aws_instance.node)) }
    ${ip} ansible_user=ubuntu
    %{ endfor }
    INV
  EOT
}
